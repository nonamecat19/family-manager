-- Notebooks. A notebook is visible to its owner, or to whoever a notebook_shares row grants
-- it to — the same "single user by default" rule the notes carry, one level up.
--
-- note_count is the CALLER's count, not the notebook's: it counts only notes that caller can
-- already see, so a notebook shared with one person does not advertise how many private notes
-- it holds by rendering a number nobody can click through to. It runs the note visibility
-- predicate from notes.sql verbatim, the notebook disjunct included — a share of THIS folder
-- reaches only the notes its owner filed here, so a note someone else parked in it is not
-- counted for anybody but its owner.

-- name: CreateNotebook :one
INSERT INTO notebooks (family_id, owner_user_id, parent_id, name)
VALUES (
    sqlc.arg('family_id')::uuid,
    sqlc.arg('owner_user_id')::uuid,
    sqlc.narg('parent_id')::uuid,
    sqlc.arg('name')::text
)
RETURNING *;

-- name: GetNotebook :one
SELECT
    nb.*,
    (SELECT count(*) FROM notes n
      WHERE n.notebook_id = nb.id
        AND NOT n.archived
        AND (
          n.owner_user_id = sqlc.arg('user_id')::uuid
          OR EXISTS (SELECT 1 FROM note_shares s
                      WHERE s.note_id = n.id
                        AND (s.subject = 2 OR s.member_user_id = sqlc.arg('user_id')::uuid))
          OR EXISTS (SELECT 1 FROM notebook_shares ns
                      JOIN notebooks nbo ON nbo.id = ns.notebook_id
                      WHERE ns.notebook_id = n.notebook_id
                        AND nbo.owner_user_id = n.owner_user_id
                        AND (ns.subject = 2 OR ns.member_user_id = sqlc.arg('user_id')::uuid))
        ))::int AS note_count,
    -- can_edit is "may the caller put a note in this notebook", which is ownership or an EDIT
    -- share on the folder. It is NOT authority over the folder itself — renaming, archiving,
    -- deleting and sharing stay the owner's, see UpdateNotebook below.
    (nb.owner_user_id = sqlc.arg('user_id')::uuid
     OR EXISTS (SELECT 1 FROM notebook_shares ns
                 WHERE ns.notebook_id = nb.id AND ns.permission = 2
                   AND (ns.subject = 2 OR ns.member_user_id = sqlc.arg('user_id')::uuid))
    )::bool AS can_edit
FROM notebooks nb
WHERE nb.id = sqlc.arg('notebook_id')::uuid
  AND nb.family_id = sqlc.arg('family_id')::uuid
  AND (
    nb.owner_user_id = sqlc.arg('user_id')::uuid
    OR EXISTS (SELECT 1 FROM notebook_shares ns
                WHERE ns.notebook_id = nb.id
                  AND (ns.subject = 2 OR ns.member_user_id = sqlc.arg('user_id')::uuid))
  );

-- name: ListNotebooks :many
SELECT
    nb.*,
    (SELECT count(*) FROM notes n
      WHERE n.notebook_id = nb.id
        AND NOT n.archived
        AND (
          n.owner_user_id = sqlc.arg('user_id')::uuid
          OR EXISTS (SELECT 1 FROM note_shares s
                      WHERE s.note_id = n.id
                        AND (s.subject = 2 OR s.member_user_id = sqlc.arg('user_id')::uuid))
          OR EXISTS (SELECT 1 FROM notebook_shares ns
                      JOIN notebooks nbo ON nbo.id = ns.notebook_id
                      WHERE ns.notebook_id = n.notebook_id
                        AND nbo.owner_user_id = n.owner_user_id
                        AND (ns.subject = 2 OR ns.member_user_id = sqlc.arg('user_id')::uuid))
        ))::int AS note_count
FROM notebooks nb
WHERE nb.family_id = sqlc.arg('family_id')::uuid
  AND (
    nb.owner_user_id = sqlc.arg('user_id')::uuid
    OR EXISTS (SELECT 1 FROM notebook_shares ns
                WHERE ns.notebook_id = nb.id
                  AND (ns.subject = 2 OR ns.member_user_id = sqlc.arg('user_id')::uuid))
  )
  AND (sqlc.arg('include_archived')::bool OR NOT nb.archived)
ORDER BY lower(btrim(nb.name)) ASC;

-- ListSharedNotebooks is the notebook half of "Shared with me": visible, but not owned.
-- name: ListSharedNotebooks :many
SELECT
    nb.*,
    (SELECT count(*) FROM notes n
      WHERE n.notebook_id = nb.id
        AND NOT n.archived
        AND (
          n.owner_user_id = sqlc.arg('user_id')::uuid
          OR EXISTS (SELECT 1 FROM note_shares s
                      WHERE s.note_id = n.id
                        AND (s.subject = 2 OR s.member_user_id = sqlc.arg('user_id')::uuid))
          OR EXISTS (SELECT 1 FROM notebook_shares ns
                      JOIN notebooks nbo ON nbo.id = ns.notebook_id
                      WHERE ns.notebook_id = n.notebook_id
                        AND nbo.owner_user_id = n.owner_user_id
                        AND (ns.subject = 2 OR ns.member_user_id = sqlc.arg('user_id')::uuid))
        ))::int AS note_count
FROM notebooks nb
WHERE nb.family_id = sqlc.arg('family_id')::uuid
  AND nb.owner_user_id <> sqlc.arg('user_id')::uuid
  AND NOT nb.archived
  AND EXISTS (SELECT 1 FROM notebook_shares ns
               WHERE ns.notebook_id = nb.id
                 AND (ns.subject = 2 OR ns.member_user_id = sqlc.arg('user_id')::uuid))
ORDER BY lower(btrim(nb.name)) ASC;

-- Only the owner may rename, re-parent or archive a notebook. An EDIT share on a notebook
-- grants writes to the notes inside it, not authority over the folder itself.
-- name: UpdateNotebook :one
UPDATE notebooks
SET name = sqlc.arg('name')::text,
    parent_id = sqlc.narg('parent_id')::uuid,
    archived = sqlc.arg('archived')::bool,
    updated_at = NOW()
WHERE id = sqlc.arg('notebook_id')::uuid
  AND family_id = sqlc.arg('family_id')::uuid
  AND owner_user_id = sqlc.arg('user_id')::uuid
RETURNING *;

-- NotebookAncestors walks parent_id upward from one notebook and returns every id on the
-- chain, the starting notebook included. UpdateNotebook uses it to refuse a re-parent that
-- would close a loop: nothing server-side recurses over the tree, but the app builds its
-- sidebar from parent_id and a cycle there is an infinite render.
--
-- It carries family_id and no visibility predicate, deliberately. A chain can run through a
-- notebook the caller cannot see — B may be parented under someone else's C — and a walk that
-- stopped at the first invisible link would report "no cycle" for a cycle that exists. What it
-- returns is ids and nothing else: no name, no owner, no count, so a caller learns only that a
-- move they asked for would close a loop, which is a fact about their own request.
--
-- The depth cap is not belt-and-braces. Nothing today stops a cycle already being in the
-- table (this query is what starts stopping it), and a recursive CTE over a cyclic graph does
-- not terminate — so the cap is what makes this statement safe to run against the data it
-- exists to clean up. 64 is far past the design's one level of nesting.
-- name: NotebookAncestors :many
WITH RECURSIVE ancestors AS (
    SELECT nb.id, nb.parent_id, 1 AS depth
    FROM notebooks nb
    WHERE nb.id = sqlc.arg('notebook_id')::uuid
      AND nb.family_id = sqlc.arg('family_id')::uuid
  UNION ALL
    SELECT parent.id, parent.parent_id, a.depth + 1
    FROM notebooks parent
    JOIN ancestors a ON parent.id = a.parent_id
    WHERE parent.family_id = sqlc.arg('family_id')::uuid
      AND a.depth < 64
)
SELECT id FROM ancestors;

-- CountNotebookNotes backs DeleteNotebook's refusal (see the rpc's comment: a mis-tapped
-- delete must not be able to take a hundred notes with it), so `total` counts EVERY note in
-- the notebook, ignoring visibility — counting only what the caller can see would let an owner
-- delete a notebook out from under notes shared into it. It is still scoped by family_id like
-- every other statement here: notes.notebook_id is a plain FK with no family constraint, and
-- the rule belongs in the SQL rather than in the handler's gates alone.
--
-- `visible` is the same set through the visibility predicate, and it is the only one of the
-- two the handler is allowed to put in a message: `total` may include notes the caller cannot
-- see, and printing that number is the existence leak note_count was written to avoid.
-- name: CountNotebookNotes :one
SELECT
    count(*)::int AS total,
    count(*) FILTER (WHERE
      n.owner_user_id = sqlc.arg('user_id')::uuid
      OR EXISTS (SELECT 1 FROM note_shares s
                  WHERE s.note_id = n.id
                    AND (s.subject = 2 OR s.member_user_id = sqlc.arg('user_id')::uuid))
      OR EXISTS (SELECT 1 FROM notebook_shares ns
                  JOIN notebooks nbo ON nbo.id = ns.notebook_id
                  WHERE ns.notebook_id = n.notebook_id
                    AND nbo.owner_user_id = n.owner_user_id
                    AND (ns.subject = 2 OR ns.member_user_id = sqlc.arg('user_id')::uuid))
    )::int AS visible
FROM notes n
WHERE n.notebook_id = sqlc.arg('notebook_id')::uuid
  AND n.family_id = sqlc.arg('family_id')::uuid;

-- name: DeleteNotebook :execrows
DELETE FROM notebooks
WHERE id = sqlc.arg('notebook_id')::uuid
  AND family_id = sqlc.arg('family_id')::uuid
  AND owner_user_id = sqlc.arg('user_id')::uuid
  AND NOT EXISTS (SELECT 1 FROM notes n
                   WHERE n.notebook_id = notebooks.id
                     AND n.family_id = notebooks.family_id);
