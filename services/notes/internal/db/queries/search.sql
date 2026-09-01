-- Search backs the ⌘K palette. Three statements, one per facet, all ranked server-side and
-- all carrying the visibility predicate from notes.sql, the notebook disjunct included (a
-- notebook share reaches its owner's notes only) — the app never filters a full note list
-- client-side, which stops working at the first notebook nobody has opened yet.
--
-- The text config is 'simple' everywhere, matching the generated notes.search column: a
-- stemmer for the wrong language is worse than no stemmer, and the two must agree or the GIN
-- index goes unused.
--
-- The breadcrumb name a hit drags along carries the notebook visibility predicate too, and not
-- only the hit's own row: a note shared on its own out of a notebook nobody shared must not
-- ship that notebook's name with it. The subselect is coalesced to '' rather than left NULL —
-- an unfiled note has no notebook at all — and the hit degrades to a bare title, because
-- breadcrumb() drops empty parts.
--
-- websearch_to_tsquery rather than plainto_tsquery, so a person can type quoted phrases and
-- OR the way they would into any other search box, and a syntactically hopeless query yields
-- an empty tsquery instead of an error.

-- SearchNotes ranks by ts_rank and returns a ts_headline snippet. StartSel/StopSel are blanked
-- because the app draws its own highlight: the server sends the matching text, not markup.
-- name: SearchNotes :many
SELECT
    n.id,
    n.notebook_id,
    n.title,
    n.owner_user_id,
    n.updated_at,
    ts_rank(n.search, websearch_to_tsquery('simple', sqlc.arg('query')::text)) AS rank,
    ts_headline('simple', n.body_text,
                websearch_to_tsquery('simple', sqlc.arg('query')::text),
                'MaxWords=18, MinWords=5, MaxFragments=1, StartSel="", StopSel=""'
    ) AS snippet,
    coalesce((SELECT nb.name FROM notebooks nb
      WHERE nb.id = n.notebook_id
        AND nb.family_id = sqlc.arg('family_id')::uuid
        AND (
          nb.owner_user_id = sqlc.arg('user_id')::uuid
          OR EXISTS (SELECT 1 FROM notebook_shares ns
                      WHERE ns.notebook_id = nb.id
                        AND (ns.subject = 2 OR ns.member_user_id = sqlc.arg('user_id')::uuid))
        )), '')::text AS notebook_name
FROM notes n
WHERE n.family_id = sqlc.arg('family_id')::uuid
  AND NOT n.archived
  AND n.search @@ websearch_to_tsquery('simple', sqlc.arg('query')::text)
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
  )
ORDER BY rank DESC, n.updated_at DESC
LIMIT NULLIF(sqlc.arg('limit_count')::int, 0);

-- SearchTasks searches the TODO blocks inside visible notes. It has to walk the block array
-- rather than the tsvector, because the tsvector flattens the whole note and cannot say which
-- line matched — and the palette's TASKS row is the line, not the note. The note-level
-- `search @@ query` guard stays in the WHERE clause anyway so the GIN index still narrows the
-- set before the array expansion runs.
-- name: SearchTasks :many
SELECT
    n.id AS note_id,
    n.notebook_id,
    n.title,
    n.updated_at,
    (b.value->>'id')::text AS block_id,
    (b.value->>'text')::text AS snippet,
    coalesce((b.value->>'checked')::bool, false)::bool AS checked,
    ts_rank(to_tsvector('simple', coalesce(b.value->>'text', '')),
            websearch_to_tsquery('simple', sqlc.arg('query')::text)) AS rank,
    coalesce((SELECT nb.name FROM notebooks nb
      WHERE nb.id = n.notebook_id
        AND nb.family_id = sqlc.arg('family_id')::uuid
        AND (
          nb.owner_user_id = sqlc.arg('user_id')::uuid
          OR EXISTS (SELECT 1 FROM notebook_shares ns
                      WHERE ns.notebook_id = nb.id
                        AND (ns.subject = 2 OR ns.member_user_id = sqlc.arg('user_id')::uuid))
        )), '')::text AS notebook_name
FROM notes n
CROSS JOIN LATERAL jsonb_array_elements(n.blocks) AS b(value)
WHERE n.family_id = sqlc.arg('family_id')::uuid
  AND NOT n.archived
  AND n.search @@ websearch_to_tsquery('simple', sqlc.arg('query')::text)
  AND b.value->>'type' IN ('3', 'BLOCK_TYPE_TODO')
  AND to_tsvector('simple', coalesce(b.value->>'text', ''))
      @@ websearch_to_tsquery('simple', sqlc.arg('query')::text)
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
  )
ORDER BY rank DESC, n.updated_at DESC
LIMIT NULLIF(sqlc.arg('limit_count')::int, 0);

-- Notebook names are one short line, so this is a substring match rather than a tsquery: a
-- person typing "road" expects "Roadmap", and no stemmer gives them that.
--
-- The match is strpos rather than LIKE. The LIKE version was parameterised and so not
-- injectable, but % and _ are metacharacters inside the pattern whichever way the pattern got
-- there: a palette query of a single "%" listed every notebook the caller can see, and "a_c"
-- silently matched "abc". strpos has no pattern language, so a typed % is a typed %.
-- name: SearchNotebooks :many
SELECT
    nb.id,
    nb.name,
    nb.parent_id,
    nb.updated_at,
    coalesce((SELECT p.name FROM notebooks p
      WHERE p.id = nb.parent_id
        AND p.family_id = sqlc.arg('family_id')::uuid
        AND (
          p.owner_user_id = sqlc.arg('user_id')::uuid
          OR EXISTS (SELECT 1 FROM notebook_shares ns
                      WHERE ns.notebook_id = p.id
                        AND (ns.subject = 2 OR ns.member_user_id = sqlc.arg('user_id')::uuid))
        )), '')::text AS parent_name
FROM notebooks nb
WHERE nb.family_id = sqlc.arg('family_id')::uuid
  AND NOT nb.archived
  AND strpos(lower(btrim(nb.name)), lower(btrim(sqlc.arg('query')::text))) > 0
  AND (
    nb.owner_user_id = sqlc.arg('user_id')::uuid
    OR EXISTS (SELECT 1 FROM notebook_shares ns
                WHERE ns.notebook_id = nb.id
                  AND (ns.subject = 2 OR ns.member_user_id = sqlc.arg('user_id')::uuid))
  )
ORDER BY lower(btrim(nb.name)) ASC
LIMIT NULLIF(sqlc.arg('limit_count')::int, 0);

-- CountVisibleNotes is what SearchResponse.searched_notes prints ("Searched 128 notes in
-- 31ms"). It is the size of the corpus the search ran over, not the number of hits.
-- name: CountVisibleNotes :one
SELECT count(*)::int AS note_count
FROM notes n
WHERE n.family_id = sqlc.arg('family_id')::uuid
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
  );
