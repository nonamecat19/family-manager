#!/usr/bin/env node
// Imports the scraped cookbook JSON (books/*/recipes/*.json) into the recipes DB.
//
// The recipes schema has no image column (services/recipes/internal/db/migrations/000001_init.up.sql
// has none, and libs/proto has no image field either), so the matching cutout .webp files under
// books/<n>/cutouts/ are NOT imported — this script only fills recipe_categories, recipes,
// recipe_ingredients and recipe_steps. Nutrition (kcal/protein/fat/carbs) has no dedicated
// column either, so it's folded into `description` as readable text.
//
// Usage:
//   FAMILY_ID=<uuid> AUTHOR_USER_ID=<uuid> \
//   DATABASE_URL=postgres://admin:root@localhost:5432/recipes?sslmode=disable \
//   node scripts/import-recipes.mjs [path/to/books]
//
// Requires `psql` on PATH. Builds one SQL file (dollar-quoted, no manual escaping needed for
// the Cyrillic/free-text fields) and runs it in a single transaction via `psql -f`.

import { readdirSync, readFileSync, mkdtempSync, writeFileSync } from "node:fs";
import { join, basename } from "node:path";
import { tmpdir } from "node:os";
import { execFileSync } from "node:child_process";

const FAMILY_ID = process.env.FAMILY_ID;
const AUTHOR_USER_ID = process.env.AUTHOR_USER_ID;
const DATABASE_URL = process.env.DATABASE_URL ?? "postgres://admin:root@localhost:5432/recipes?sslmode=disable";
const booksDir = process.argv[2] ?? join(process.env.HOME, "temp/recepies/books");

if (!FAMILY_ID || !AUTHOR_USER_ID) {
  console.error("FAMILY_ID and AUTHOR_USER_ID env vars are required (recipes rows are NOT NULL on both).");
  process.exit(1);
}

const UUID_TAG = "zzimport"; // dollar-quote tag unlikely to collide with recipe text
const dq = (s) => `$${UUID_TAG}$${s}$${UUID_TAG}$`;

function loadRecipes(dir) {
  const bookDirs = readdirSync(dir, { withFileTypes: true }).filter((d) => d.isDirectory());
  const out = [];
  for (const b of bookDirs) {
    const recipesDir = join(dir, b.name, "recipes");
    let files = [];
    try {
      files = readdirSync(recipesDir).filter((f) => f.endsWith(".json"));
    } catch {
      continue; // book has no recipes/ dir or it's empty (e.g. book 1)
    }
    for (const f of files) {
      const raw = JSON.parse(readFileSync(join(recipesDir, f), "utf8"));
      out.push(raw);
    }
  }
  return out;
}

const recipes = loadRecipes(booksDir);
if (recipes.length === 0) {
  console.error(`No recipe JSON files found under ${booksDir}`);
  process.exit(1);
}

function describeNutrition(r) {
  const n = r.nutrition_per_serving;
  const parts = [];
  if (r.time_raw) parts.push(`Час: ${r.time_raw}`);
  if (n) {
    parts.push(
      `На порцію: ${n.kcal ?? "?"} ккал, білки ${n.protein_g ?? "?"} г, жири ${n.fat_g ?? "?"} г, вуглеводи ${n.carbs_g ?? "?"} г`,
    );
  }
  return parts.join(". ");
}

const sql = [];
sql.push("BEGIN;");
sql.push(`\\set family_id ${FAMILY_ID}`);
sql.push(`\\set author_id ${AUTHOR_USER_ID}`);

// Upsert categories by (family_id, name) first, keyed off `section`, then reuse via subselect
// per recipe insert rather than round-tripping ids through psql variables.
const sections = [...new Set(recipes.map((r) => r.section).filter(Boolean))];
for (const name of sections) {
  sql.push(
    `INSERT INTO recipe_categories (family_id, name) VALUES (:'family_id', ${dq(name)})\n` +
      `  ON CONFLICT (family_id, lower(btrim(name))) DO NOTHING;`,
  );
}

let skipped = 0;
for (const r of recipes) {
  if (!r.title || !r.title.trim()) {
    skipped++;
    continue;
  }
  const timeMinutes = Number.isFinite(r.time_minutes) ? r.time_minutes : 0;
  const cookSeconds = Math.max(0, Math.round(timeMinutes * 60));
  const description = describeNutrition(r);
  const categoryExpr = r.section
    ? `(SELECT id FROM recipe_categories WHERE family_id = :'family_id' AND lower(btrim(name)) = lower(btrim(${dq(r.section)})))`
    : "NULL";

  sql.push(
    `WITH new_recipe AS (\n` +
      `  INSERT INTO recipes (family_id, title, description, category_id, prep_seconds, cook_seconds, author_user_id)\n` +
      `  VALUES (:'family_id', ${dq(r.title)}, ${dq(description)}, ${categoryExpr}, 0, ${cookSeconds}, :'author_id')\n` +
      `  RETURNING id\n` +
      `)`,
  );

  const ingredientRows = (r.ingredients ?? [])
    .map((ing, i) => {
      const amount = ing.amount != null ? String(ing.amount) : "";
      return `(${i}, ${dq(ing.name ?? "")}, ${dq(amount)}, ${dq(ing.unit ?? "")})`;
    })
    .join(",\n    ");

  const stepRows = (r.steps ?? [])
    .map((step, i) => `(${i}, ${dq(step)})`)
    .join(",\n    ");

  const inserts = [];
  if (ingredientRows) {
    inserts.push(
      `ins_ingredients AS (\n` +
        `  INSERT INTO recipe_ingredients (recipe_id, position, name, amount, unit)\n` +
        `  SELECT new_recipe.id, v.position, v.name, v.amount, v.unit FROM new_recipe,\n` +
        `    (VALUES\n    ${ingredientRows}\n    ) AS v(position, name, amount, unit)\n` +
        `  RETURNING 1\n` +
        `)`,
    );
  }
  if (stepRows) {
    inserts.push(
      `ins_steps AS (\n` +
        `  INSERT INTO recipe_steps (recipe_id, position, instruction)\n` +
        `  SELECT new_recipe.id, v.position, v.instruction FROM new_recipe,\n` +
        `    (VALUES\n    ${stepRows}\n    ) AS v(position, instruction)\n` +
        `  RETURNING 1\n` +
        `)`,
    );
  }

  if (inserts.length === 0) {
    sql[sql.length - 1] += "\nSELECT id FROM new_recipe;";
  } else {
    const lastCte = sql.pop();
    sql.push(`${lastCte},\n${inserts.join(",\n")}\nSELECT id FROM new_recipe;`);
  }
}

sql.push("COMMIT;");

const tmpFile = join(mkdtempSync(join(tmpdir(), "recipes-import-")), "import.sql");
writeFileSync(tmpFile, sql.join("\n\n"));

console.log(`Recipes found: ${recipes.length}, skipped (no title): ${skipped}, categories: ${sections.length}`);
console.log(`SQL written to ${tmpFile}`);

execFileSync("psql", [DATABASE_URL, "-v", "ON_ERROR_STOP=1", "-f", tmpFile], { stdio: "inherit" });

console.log("Import complete.");
