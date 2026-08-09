#!/usr/bin/env node
// Imports the scraped cookbook JSON (books/*/recipes/*.json) into the recipes DB, and
// uploads each recipe's matching cutout image (books/<n>/cutouts/recipe-<page>.webp) to
// MinIO via the `aws` CLI (S3-compatible), setting recipes.image_url to the public URL.
// Nutrition (kcal/protein/fat/carbs) has no dedicated column, so it's folded into
// `description` as readable text.
//
// Each book is a meal of the day (1 breakfast, 2 lunch, 3 dinner) — that's the category.
// `section` (e.g. "Wok-обіди") is a subcategory within that meal, not a category of its own.
//
// Usage:
//   FAMILY_ID=<uuid> AUTHOR_USER_ID=<uuid> \
//   DATABASE_URL=postgres://admin:root@localhost:5432/recipes?sslmode=disable \
//   MINIO_ENDPOINT=http://localhost:9000 MINIO_PUBLIC_URL=http://localhost:9000 \
//   MINIO_BUCKET=recipes AWS_ACCESS_KEY_ID=minioadmin AWS_SECRET_ACCESS_KEY=minioadmin \
//   node scripts/import-recipes.mjs [path/to/books]
//
// Requires `psql` and `aws` on PATH. If `aws` is missing or MinIO is unreachable, image
// upload is skipped (with a warning) and recipes import with image_url = ''. Builds one SQL
// file (dollar-quoted, no manual escaping needed for the Cyrillic/free-text fields) and runs
// it in a single transaction via `psql -f`.

import { readdirSync, readFileSync, mkdtempSync, writeFileSync, existsSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { execFileSync } from "node:child_process";

const FAMILY_ID = process.env.FAMILY_ID;
const AUTHOR_USER_ID = process.env.AUTHOR_USER_ID;
const DATABASE_URL = process.env.DATABASE_URL ?? "postgres://admin:root@localhost:5432/recipes?sslmode=disable";
const MINIO_ENDPOINT = process.env.MINIO_ENDPOINT ?? "http://localhost:9000";
const MINIO_PUBLIC_URL = process.env.MINIO_PUBLIC_URL ?? MINIO_ENDPOINT;
const MINIO_BUCKET = process.env.MINIO_BUCKET ?? "recipes";
const booksDir = process.argv[2] ?? join(process.env.HOME, "temp/recepies/books");

if (!FAMILY_ID || !AUTHOR_USER_ID) {
  console.error("FAMILY_ID and AUTHOR_USER_ID env vars are required (recipes rows are NOT NULL on both).");
  process.exit(1);
}

const UUID_TAG = "zzimport"; // dollar-quote tag unlikely to collide with recipe text
const dq = (s) => `$${UUID_TAG}$${s}$${UUID_TAG}$`;

// Each book is a meal of the day, not a topic — book 1 is breakfast, 2 is lunch, 3 is
// dinner. That's the category; `section` (e.g. "Wok-обіди") is a subcategory *within* that
// meal, not a category of its own.
const BOOK_CATEGORIES = { 1: "Сніданок", 2: "Обід", 3: "Вечеря" };

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
      raw._cutoutPath = join(dir, b.name, "cutouts", `recipe-${String(raw.page).padStart(3, "0")}.webp`);
      out.push(raw);
    }
  }
  return out;
}

// Uploads via the aws CLI against MinIO's S3-compatible API. Returns null (not throw) on any
// failure so a broken/unreachable MinIO degrades to "no images" rather than aborting the
// whole import — the recipe text is the valuable part.
let awsAvailable = true;
function uploadImage(localPath, key) {
  if (!awsAvailable || !existsSync(localPath)) return null;
  try {
    execFileSync(
      "aws",
      ["--endpoint-url", MINIO_ENDPOINT, "s3", "cp", localPath, `s3://${MINIO_BUCKET}/${key}`],
      {
        env: {
          ...process.env,
          AWS_ACCESS_KEY_ID: process.env.AWS_ACCESS_KEY_ID ?? "minioadmin",
          AWS_SECRET_ACCESS_KEY: process.env.AWS_SECRET_ACCESS_KEY ?? "minioadmin",
        },
        stdio: "ignore",
      },
    );
    return `${MINIO_PUBLIC_URL}/${MINIO_BUCKET}/${key}`;
  } catch (err) {
    console.warn(`Image upload disabled after failure on ${localPath}: ${err.message}`);
    awsAvailable = false;
    return null;
  }
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

// Upsert categories (one per book/meal) first, then subcategories (one per book+section pair,
// since the same section name could in principle appear under two meals), reusing both via
// subselects per recipe insert rather than round-tripping ids through psql variables.
const books = [...new Set(recipes.map((r) => r.book))];
for (const book of books) {
  const name = BOOK_CATEGORIES[book] ?? `Книга ${book}`;
  sql.push(
    `INSERT INTO recipe_categories (family_id, name) VALUES (:'family_id', ${dq(name)})\n` +
      `  ON CONFLICT (family_id, lower(btrim(name))) DO NOTHING;`,
  );
}

const subcategoryMap = new Map();
for (const r of recipes) {
  if (!r.section) continue;
  subcategoryMap.set(`${r.book}|${r.section}`, { book: r.book, section: r.section });
}
for (const { book, section } of subcategoryMap.values()) {
  const categoryName = BOOK_CATEGORIES[book] ?? `Книга ${book}`;
  sql.push(
    `INSERT INTO recipe_subcategories (category_id, family_id, name)\n` +
      `  SELECT id, :'family_id', ${dq(section)} FROM recipe_categories\n` +
      `  WHERE family_id = :'family_id' AND lower(btrim(name)) = lower(btrim(${dq(categoryName)}))\n` +
      `  ON CONFLICT (category_id, lower(btrim(name))) DO NOTHING;`,
  );
}

let skipped = 0;
let imagesUploaded = 0;
for (const r of recipes) {
  if (!r.title || !r.title.trim()) {
    skipped++;
    continue;
  }
  const timeMinutes = Number.isFinite(r.time_minutes) ? r.time_minutes : 0;
  const cookSeconds = Math.max(0, Math.round(timeMinutes * 60));
  const description = describeNutrition(r);
  const categoryName = BOOK_CATEGORIES[r.book] ?? `Книга ${r.book}`;
  const categoryExpr =
    `(SELECT id FROM recipe_categories WHERE family_id = :'family_id' ` +
    `AND lower(btrim(name)) = lower(btrim(${dq(categoryName)})))`;
  const subcategoryExpr = r.section
    ? `(SELECT s.id FROM recipe_subcategories s JOIN recipe_categories c ON c.id = s.category_id\n` +
      `    WHERE s.family_id = :'family_id' AND c.family_id = :'family_id'\n` +
      `    AND lower(btrim(c.name)) = lower(btrim(${dq(categoryName)}))\n` +
      `    AND lower(btrim(s.name)) = lower(btrim(${dq(r.section)})))`
    : "NULL";

  const imageKey = `${FAMILY_ID}/book${r.book}-page${String(r.page).padStart(3, "0")}.webp`;
  const imageUrl = uploadImage(r._cutoutPath, imageKey);
  if (imageUrl) imagesUploaded++;

  sql.push(
    `WITH new_recipe AS (\n` +
      `  INSERT INTO recipes (family_id, title, description, category_id, subcategory_id, prep_seconds, cook_seconds, author_user_id, image_url)\n` +
      `  VALUES (:'family_id', ${dq(r.title)}, ${dq(description)}, ${categoryExpr}, ${subcategoryExpr}, 0, ${cookSeconds}, :'author_id', ${dq(imageUrl ?? "")})\n` +
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

console.log(
  `Recipes found: ${recipes.length}, skipped (no title): ${skipped}, categories: ${books.length}, ` +
    `subcategories: ${subcategoryMap.size}, images uploaded: ${imagesUploaded}`,
);
console.log(`SQL written to ${tmpFile}`);

execFileSync("psql", [DATABASE_URL, "-v", "ON_ERROR_STOP=1", "-f", tmpFile], { stdio: "inherit" });

console.log("Import complete.");
