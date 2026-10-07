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

-- name: SumIngredientsForBasket :many
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
