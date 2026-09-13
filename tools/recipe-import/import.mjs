#!/usr/bin/env node
import { createHash, randomUUID } from "node:crypto";
import { DatabaseSync } from "node:sqlite";
import { mkdirSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";


function parseArgs(argv) {
  const out = {};
  for (let i = 0; i < argv.length; i += 2) {
    const k = argv[i];
    if (!k.startsWith("--")) throw new Error(`expected a --flag, got ${k}`);
    out[k.slice(2)] = argv[i + 1];
  }
  return out;
}

const args = parseArgs(process.argv.slice(2));
const REQUIRED = ["db", "books", "family", "author", "out-sql", "out-images"];
for (const k of REQUIRED) {
  if (args[k] === undefined || args[k] === "") {
    console.error(`missing --${k}\n\nusage: see the header of ${import.meta.filename}`);
    process.exit(2);
  }
}
const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
for (const k of ["family", "author"]) {
  if (!UUID_RE.test(args[k])) {
    console.error(`--${k} must be a uuid, got ${JSON.stringify(args[k])}`);
    process.exit(2);
  }
}
const familyId = args.family.toLowerCase();
const authorId = args.author.toLowerCase();
const publicUrl = (args["public-url"] ?? "").replace(/\/+$/, "");


const NAMESPACE = "6f9c1f2e-9a3b-5d47-8f21-2c7a4b6e0d13";

function uuid5(namespace, name) {
  const nsBytes = Buffer.from(namespace.replace(/-/g, ""), "hex");
  const hash = createHash("sha1").update(nsBytes).update(Buffer.from(name, "utf8")).digest();
  const b = Buffer.from(hash.subarray(0, 16));
  b[6] = (b[6] & 0x0f) | 0x50;
  b[8] = (b[8] & 0x3f) | 0x80;
  const h = b.toString("hex");
  return `${h.slice(0, 8)}-${h.slice(8, 12)}-${h.slice(12, 16)}-${h.slice(16, 20)}-${h.slice(20)}`;
}

const idFor = (kind, key) => uuid5(NAMESPACE, `${familyId}|${kind}|${key}`);


const lit = (s) => `'${String(s).replace(/'/g, "''")}'`;
const num = (n) => (n === null || n === undefined || Number.isNaN(n) ? "0" : String(n));


const PREP_STATE_UK = {
  dry: "у сухому вигляді",
  cooked: "у відвареному вигляді",
  baked: "у запеченому вигляді",
  fried: "у смаженому вигляді",
  canned: "у консервованому вигляді",
  dried: "у сушеному вигляді",
  frozen: "у замороженому вигляді",
  smoked: "у копченому вигляді",
  lightly_salted: "у слабосоленому вигляді",
  pickled: "у маринованому вигляді",
  fresh: "у свіжому вигляді",
  ground: "у меленому вигляді",
  peeled: "в очищеному вигляді",
  ready_made: "у готовому вигляді",
};

function ingredientName(row) {
  let name = row.name_uk;
  if (row.fat_pct !== null) {
    const pct = Number.isInteger(row.fat_pct) ? row.fat_pct : row.fat_pct.toFixed(1);
    name += ` ${pct}%`;
  }
  if (row.prep_state !== null) {
    name += ` (${PREP_STATE_UK[row.prep_state] ?? row.prep_state})`;
  }
  return name;
}

function ingredientAmount(row) {
  if (row.grams !== null) {
    const g = Number.isInteger(row.grams) ? row.grams : Number(row.grams.toFixed(1));
    return { amount: String(g), unit: "г" };
  }
  return { amount: "", unit: "за смаком" };
}

function splitProse(text) {
  const parts = text
    .split(/(?<=[.!?])\s+(?=[A-ZА-ЯЁЇІЄҐ])/u)
    .map((s) => s.trim())
    .filter((s) => s !== "");
  if (parts.length < 2 || parts.some((p) => p.length < 20)) return [text.trim()];
  return parts;
}

const COMPONENT_LABEL = { sauce: "Соус", salad: "Салат", dressing: "Заправка" };

function buildNotes(r) {
  const lines = [];
  const add = (label, value) => {
    if (value !== null && value !== undefined && String(value).trim() !== "") {
      lines.push(`${label}: ${String(value).trim()}`);
    }
  };
  add("Подача", r.serving);
  add("Порада", r.note);
  add("Варіація", r.variation);
  add("Про інгредієнти", r.ingredients_note);
  add("Про поживність", r.nutrition_addon);
  if (r.chill_minutes !== null) {
    add("Витримка", `${r.chill_minutes} хв (не входить у час приготування)`);
  }
  add("Про джерело", r.source_note);
  lines.push(`Джерело: книга ${r.book_id}, с. ${r.page}`);
  return lines.join("\n");
}


const db = new DatabaseSync(args.db, { readOnly: true });
const q = (sql, ...p) => db.prepare(sql).all(...p);

const books = q("SELECT id, title FROM book ORDER BY id");
const sections = q("SELECT id, book_id, name FROM section ORDER BY id");
const recipes = q(`
  SELECT r.*, s.book_id AS section_book_id
  FROM recipe r JOIN section s ON s.id = r.section_id
  ORDER BY r.id`);
const stepsByRecipe = new Map();
for (const s of q("SELECT recipe_id, position, instruction FROM recipe_step ORDER BY recipe_id, position")) {
  if (!stepsByRecipe.has(s.recipe_id)) stepsByRecipe.set(s.recipe_id, []);
  stepsByRecipe.get(s.recipe_id).push(s);
}
const componentsByRecipe = new Map();
for (const c of q("SELECT recipe_id, kind, instruction FROM recipe_component ORDER BY recipe_id, kind")) {
  if (!componentsByRecipe.has(c.recipe_id)) componentsByRecipe.set(c.recipe_id, []);
  componentsByRecipe.get(c.recipe_id).push(c);
}
const ingByRecipe = new Map();
for (const ri of q(`
  SELECT ri.recipe_id, ri.position, ri.grams, ri.prep_state, ri.fat_pct, i.name_uk
  FROM recipe_ingredient ri JOIN ingredient i ON i.id = ri.ingredient_id
  ORDER BY ri.recipe_id, ri.position`)) {
  if (!ingByRecipe.has(ri.recipe_id)) ingByRecipe.set(ri.recipe_id, []);
  ingByRecipe.get(ri.recipe_id).push(ri);
}


const sql = [];
const images = [];
const stats = { categories: 0, subcategories: 0, recipes: 0, skippedDuplicates: 0, steps: 0, ingredients: 0, prosePlit: 0 };

sql.push("-- Generated by tools/recipe-import/import.mjs. Do not edit by hand.");
sql.push(`-- family_id ${familyId}, author_user_id ${authorId}`);
sql.push("--");
sql.push("-- Idempotent: every id is a UUIDv5 of the family id and the source row, and every");
sql.push("-- INSERT is ON CONFLICT DO UPDATE, so re-running refreshes the rows in place rather");
sql.push("-- than duplicating them or failing.");
sql.push("BEGIN;");
sql.push("");

sql.push("-- book -> recipe_categories");
for (const b of books) {
  const id = idFor("category", `book:${b.id}`);
  stats.categories++;
  sql.push(
    `INSERT INTO recipe_categories (id, family_id, name) VALUES (${lit(id)}, ${lit(familyId)}, ${lit(b.title)})\n` +
      `  ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name;`,
  );
}
sql.push("");

sql.push("-- section -> recipe_subcategories");
for (const s of sections) {
  const id = idFor("subcategory", `section:${s.id}`);
  const catId = idFor("category", `book:${s.book_id}`);
  stats.subcategories++;
  sql.push(
    `INSERT INTO recipe_subcategories (id, category_id, family_id, name)\n` +
      `  VALUES (${lit(id)}, ${lit(catId)}, ${lit(familyId)}, ${lit(s.name)})\n` +
      `  ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, category_id = EXCLUDED.category_id;`,
  );
}
sql.push("");

sql.push("-- recipe -> recipes (+ ingredients, steps)");
for (const r of recipes) {
  if (r.duplicate_of_recipe_id !== null) {
    stats.skippedDuplicates++;
    continue;
  }

  const id = idFor("recipe", `recipe:${r.id}`);
  const catId = idFor("category", `book:${r.section_book_id}`);
  const subId = idFor("subcategory", `section:${r.section_id}`);

  let imageUrl = "";
  if (r.cutout_image !== null && publicUrl !== "") {
    const key = `${familyId}/${id}.webp`;
    imageUrl = `${publicUrl}/${key}`;
    images.push(`${join(args.books, r.cutout_image)}\t${key}`);
  }

  const cookSeconds = (r.time_minutes ?? 0) * 60;

  stats.recipes++;
  sql.push(
    `INSERT INTO recipes (id, family_id, title, description, category_id, subcategory_id,\n` +
      `    servings, prep_seconds, cook_seconds, author_user_id, notes, rating, image_url,\n` +
      `    kcal, protein_g, fat_g, carbs_g)\n` +
      `  VALUES (${lit(id)}, ${lit(familyId)}, ${lit(r.title)}, '', ${lit(catId)}, ${lit(subId)},\n` +
      `    1, 0, ${num(cookSeconds)}, ${lit(authorId)}, ${lit(buildNotes(r))}, 0, ${lit(imageUrl)},\n` +
      `    ${num(r.kcal)}, ${num(r.protein_g)}, ${num(r.fat_g)}, ${num(r.carbs_g)})\n` +
      `  ON CONFLICT (id) DO UPDATE SET\n` +
      `    title = EXCLUDED.title, category_id = EXCLUDED.category_id,\n` +
      `    subcategory_id = EXCLUDED.subcategory_id, cook_seconds = EXCLUDED.cook_seconds,\n` +
      `    notes = EXCLUDED.notes, image_url = EXCLUDED.image_url, kcal = EXCLUDED.kcal,\n` +
      `    protein_g = EXCLUDED.protein_g, fat_g = EXCLUDED.fat_g, carbs_g = EXCLUDED.carbs_g,\n` +
      `    updated_at = NOW();`,
  );

  sql.push(`DELETE FROM recipe_ingredients WHERE recipe_id = ${lit(id)};`);
  const ings = ingByRecipe.get(r.id) ?? [];
  ings.forEach((ri, i) => {
    const { amount, unit } = ingredientAmount(ri);
    stats.ingredients++;
    sql.push(
      `INSERT INTO recipe_ingredients (recipe_id, position, name, amount, unit)\n` +
        `  VALUES (${lit(id)}, ${i}, ${lit(ingredientName(ri))}, ${lit(amount)}, ${lit(unit)});`,
    );
  });

  const numbered = stepsByRecipe.get(r.id) ?? [];
  let instructions;
  if (numbered.length > 0) {
    instructions = numbered.map((s) => s.instruction);
  } else if (r.method_summary !== null && r.method_summary.trim() !== "") {
    instructions = splitProse(r.method_summary);
    if (instructions.length > 1) stats.prosePlit++;
  } else {
    instructions = [];
  }
  for (const c of componentsByRecipe.get(r.id) ?? []) {
    instructions.push(`${COMPONENT_LABEL[c.kind] ?? c.kind}: ${c.instruction}`);
  }

  sql.push(`DELETE FROM recipe_steps WHERE recipe_id = ${lit(id)};`);
  instructions.forEach((text, i) => {
    stats.steps++;
    sql.push(
      `INSERT INTO recipe_steps (recipe_id, position, instruction, duration_seconds)\n` +
        `  VALUES (${lit(id)}, ${i + 1}, ${lit(text)}, 0);`,
    );
  });
  sql.push("");
}

sql.push("COMMIT;");
sql.push("");


for (const [path, body] of [
  [args["out-sql"], sql.join("\n")],
  [args["out-images"], images.join("\n") + (images.length > 0 ? "\n" : "")],
]) {
  mkdirSync(dirname(path), { recursive: true });
  writeFileSync(path, body);
}

console.error(
  [
    `categories        ${stats.categories}`,
    `subcategories     ${stats.subcategories}`,
    `recipes           ${stats.recipes}  (skipped ${stats.skippedDuplicates} reprint${stats.skippedDuplicates === 1 ? "" : "s"})`,
    `ingredient rows   ${stats.ingredients}`,
    `step rows         ${stats.steps}  (${stats.prosePlit} recipes split from prose)`,
    `images to upload  ${images.length}`,
    ``,
    `sql    -> ${args["out-sql"]}`,
    `images -> ${args["out-images"]}`,
  ].join("\n"),
);

