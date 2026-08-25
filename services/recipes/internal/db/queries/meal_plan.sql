-- name: PlanMeal :one
INSERT INTO meal_plan_entries (family_id, recipe_id, plan_date, slot, servings)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (family_id, plan_date, slot) DO UPDATE
    SET recipe_id = EXCLUDED.recipe_id, servings = EXCLUDED.servings
RETURNING *;

-- name: ListMealPlan :many
SELECT * FROM meal_plan_entries
WHERE family_id = $1
  AND plan_date >= $2
  AND plan_date <= $3
ORDER BY plan_date, slot;

-- name: RemoveMealPlanEntry :execrows
DELETE FROM meal_plan_entries
WHERE id = $1 AND family_id = $2;

-- TotalIngredients: aggregates every ingredient across the meal plan range, scaled by each
-- entry's servings relative to its recipe's servings. Sums by name+unit so "flour / g" from
-- two recipes adds to one line on the shopping list. The amount is summed as numeric text
-- (recipes store free-form amounts); non-numeric amounts are summed as count (1 per row) so
-- "2 cloves" + "3 cloves" becomes "2" — the app shows the breakdown for non-numeric totals.
-- name: TotalIngredients :many
WITH scaled AS (
    SELECT
        lower(btrim(ri.name)) AS name,
        lower(btrim(ri.unit)) AS unit,
        CASE
            WHEN m.servings = 0 THEN 1.0
            ELSE m.servings::numeric / NULLIF(r.servings, 0)
        END AS scale,
        ri.amount
    FROM meal_plan_entries m
    JOIN recipes r ON r.id = m.recipe_id
    JOIN recipe_ingredients ri ON ri.recipe_id = m.recipe_id
    WHERE m.family_id = $1
      AND m.plan_date >= $2
      AND m.plan_date <= $3
)
SELECT
    name,
    unit,
    COALESCE(
        SUM(CASE
            WHEN amount ~ '^[0-9]+(\.[0-9]+)?$'
            THEN amount::numeric * scale
            ELSE 1
        END),
        0
    )::text AS total_amount
FROM scaled
GROUP BY name, unit
ORDER BY name, unit;

-- SumIngredientsForBasket answers the ad-hoc question ("I plan to cook these, what do I
-- buy") without persisting anything: the basket arrives as two parallel arrays and is
-- unnested into rows. Same scaling and same name+unit grouping as TotalIngredients, so the
-- calendar and the basket produce identical lines for identical input. The family_id join
-- condition is what stops a caller totalling another family's recipes by id.
-- name: SumIngredientsForBasket :many
-- The two arrays are unnested separately and re-joined on ordinality rather than with the
-- two-argument unnest(a, b) form, which sqlc's query analyser cannot type.
WITH ids AS (
    SELECT t.recipe_id, t.ord
    FROM unnest(@recipe_ids::uuid[]) WITH ORDINALITY AS t(recipe_id, ord)
), servs AS (
    SELECT t.servings, t.ord
    FROM unnest(@servings_list::int[]) WITH ORDINALITY AS t(servings, ord)
), basket AS (
    SELECT ids.recipe_id, servs.servings
    FROM ids JOIN servs ON servs.ord = ids.ord
), scaled AS (
    SELECT
        lower(btrim(ri.name)) AS name,
        lower(btrim(ri.unit)) AS unit,
        CASE
            WHEN b.servings = 0 THEN 1.0
            ELSE b.servings::numeric / NULLIF(r.servings, 0)
        END AS scale,
        ri.amount
    FROM basket b
    JOIN recipes r ON r.id = b.recipe_id AND r.family_id = @family_id
    JOIN recipe_ingredients ri ON ri.recipe_id = r.id
)
SELECT
    name,
    unit,
    COALESCE(
        SUM(CASE
            WHEN amount ~ '^[0-9]+(\.[0-9]+)?$'
            THEN amount::numeric * scale
            ELSE 1
        END),
        0
    )::text AS total_amount
FROM scaled
GROUP BY name, unit
ORDER BY name, unit;
