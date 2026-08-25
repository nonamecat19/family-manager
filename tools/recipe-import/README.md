# recipe-import

Loads the normalized Ukrainian recipe books (3 books, 459 recipes) into `services/recipes`.

Two steps, because the rows and the photos go to different places:

```sh
# 1. rows -> SQL, photos -> a manifest
node tools/recipe-import/import.mjs \
  --db         ~/temp/recepies/books/normalize/out/recipes.db \
  --books      ~/temp/recepies/books \
  --family     <family uuid> \
  --author     <user uuid> \
  --public-url https://<r2 public host> \
  --out-sql    out/seed.sql \
  --out-images out/images.tsv

# 2. photos -> object storage (keys must match, so run this with the same --family)
tools/recipe-import/upload-images.sh --env-file /opt/family-manager/.env out/images.tsv

# 3. rows -> the database
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f out/seed.sql
```

`--family` and `--author` are a real family and a real user: this cookbook lands in someone's
account, and `family_id` is what every read is scoped by. Register in the app first, then take
the ids from the `family` and `auth` databases.

## Idempotent

Every id is a UUIDv5 of the family id and the source row, and every insert is
`ON CONFLICT DO UPDATE`. Re-running refreshes the rows in place — it does not duplicate them,
and the uploaded image keys still line up with the recipes. Ingredients and steps are deleted
and re-inserted per recipe, so rows the source drops do not linger.

Because the ids depend on the family id, seeding a *different* family produces a completely
separate set of rows. That is the correct behaviour, but it also means you cannot move a
seeded cookbook between families by re-running the import.

## The mapping

| source | target |
|---|---|
| `book` (3) | `recipe_categories` |
| `section` (35) | `recipe_subcategories` |
| `recipe` (459, 2 reprints skipped) | `recipes` |
| `recipe_step` where printed, else `method_summary` split into sentences | `recipe_steps` |
| `recipe_component` (sauce/salad/dressing) | trailing `recipe_steps` |
| `recipe_ingredient` + `ingredient` | `recipe_ingredients` |
| `kcal`, `protein_g`, `fat_g`, `carbs_g` | `recipes.*` (migration `000004_nutrition`) |
| `serving`, `note`, `variation`, `source_note`, `ingredients_note`, `nutrition_addon`, `chill_minutes`, book+page | `recipes.notes`, as labelled lines |
| `cutout_image` | object storage, `recipes.image_url` |

Judgement calls worth knowing, each argued at its call site in `import.mjs`:

* **Ingredients are stored in grams**, not in the printed unit. `TotalIngredients` aggregates
  the shopping list by name + unit, and "3 шт." and "180 г" of the same egg would not add up.
  The source guarantees a gram weight on every row that is not "to taste".
* **Preparation state and fat percentage are part of the ingredient name**
  (`Кіноа (у сухому вигляді)`, `Йогурт грецький 10%`). Dry and cooked quinoa weigh very
  differently for the same portion, so merging them would produce a wrong total.
* **Prose methods are split into sentences.** 314 recipes print prose rather than numbered
  steps, and the cook screen walks steps one at a time; a single giant step would make cook
  mode useless for two thirds of the cookbook. The split is conservative and falls back to one
  step whenever it would produce an implausible fragment.
* **Total time goes to `cook_seconds`, prep stays 0.** The books print one number. Everything
  that reads it uses `prep + cook`.
* **Chilling time is a note, not cook time.** "What can I make quickly" is about active time;
  an overnight soak would otherwise hide a 10-minute recipe behind an 8-hour filter.
* **Per-step timers are 0.** The books time the recipe, not the step, and parsing "12–14 хв"
  out of the prose would put numbers on the cook screen that the book never claimed.
* **Servings is 1.** Nutrition in these books is per serving and the recipes are written as
  single portions.

Dropped, with reasons, at the bottom of `import.mjs`: ingredient groups (15 rows), ingredient
alternatives, loose qualifiers, the printed amount form, and the full-page scans (1.8 GB — only
the background-removed dish cutouts are uploaded).
