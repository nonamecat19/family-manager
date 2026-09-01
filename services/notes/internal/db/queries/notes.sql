-- Notes: the reads here all carry the visibility predicate, and the writes all carry the
-- family scope. The predicate is repeated in full in every read rather than hidden in a view,
-- because a view is a place a future query can forget to use; a copy that is right in every
-- query is worth more than a definition that is right in one place and optional.
--
-- THE RULE, once, in prose. A note is visible to :user_id when they own it, OR a note_shares
-- row grants it to them (directly or to the whole family), OR a notebook_shares row grants
-- its notebook to them AND THAT NOTEBOOK BELONGS TO THE NOTE'S OWNER — always ANDed with
-- family_id = :family_id.
--
-- The clause in capitals is the whole share model, and it is why every notebook_shares
-- disjunct in this service joins notebooks nbo and requires nbo.owner_user_id =
-- <note>.owner_user_id. A notebook share carries the notes ITS OWNER put in it and nothing
-- else: "share my notebook" means "share the notes I keep in it", which is what a person
-- means by it. Without the clause a notebook share reaches every note in the folder no matter
-- who wrote it, and two things follow that no user would accept:
--
--   * Bob files a note into Alice's notebook (he holds an EDIT share on it). Alice shares that
--     notebook with the family, and Bob's private note is now readable by everyone — writable
--     too, if the share is EDIT.
--   * Alice, the notebook's owner, grants HERSELF read and write over Bob's note in one call:
--     ShareNotebookWithFamily makes ns.subject = 2 true for every member, herself included.
--
-- A share is a grant over what you own; it can never hand out someone else's note. The rule is
-- therefore written the same way in every place the notebook disjunct appears — the read and
-- edit predicates and the `shared` flag here, note_count and CountNotebookNotes
-- (notebooks.sql), the three search statements (search.sql), comments and activity
-- (comments_activity.sql) and ListNoteShares (shares.sql). Those files say "the notebook
-- disjunct, from notes.sql" rather than restating it; this is where it is defined.
--
-- can_edit is the same walk with permission = 2 (EDIT) added, and it is returned as a column
-- on every note read so the handler never re-derives the rule in Go.
--
-- Task counts read the block array: a TODO block is BlockType 3. The comparison is textual and
-- accepts both encodings because a JSON enum can arrive either as its number or as its proto
-- name depending on how the handler marshals a Block.
--
-- Every statement here lists the notes columns explicitly instead of using * (the rest of this
-- repo uses *). The reason is `search`: it is a generated tsvector that nothing in Go ever
-- reads, and selecting it types the field as interface{} in the generated model — an untyped
-- value pgx has no registered codec for, sitting in the middle of every note. Naming the
-- columns keeps it out of the row and off the wire.

-- name: CreateNote :one
INSERT INTO notes (
    family_id, owner_user_id, notebook_id, client_id, title, blocks, body_text,
    last_edited_by_user_id
) VALUES (
    sqlc.arg('family_id')::uuid,
    sqlc.arg('owner_user_id')::uuid,
    sqlc.narg('notebook_id')::uuid,
    sqlc.arg('client_id')::text,
    sqlc.arg('title')::text,
    sqlc.arg('blocks')::jsonb,
    sqlc.arg('body_text')::text,
    sqlc.arg('owner_user_id')::uuid
)
RETURNING
    id, family_id, owner_user_id, notebook_id, client_id, title, blocks, body_text,
    starred, archived, version, last_edited_by_user_id, created_at, updated_at;

-- CreateNoteByClientID is the offline-capture path. The conflict target names the partial
-- unique index's predicate so Postgres picks that index as the arbiter; DO UPDATE with a
-- no-op assignment (rather than DO NOTHING) is what makes the statement return the row the
-- FIRST call created, which is the whole point — a retried create answers with the original
-- note instead of a duplicate or an empty result.
-- name: CreateNoteByClientID :one
INSERT INTO notes (
    family_id, owner_user_id, notebook_id, client_id, title, blocks, body_text,
    last_edited_by_user_id
) VALUES (
    sqlc.arg('family_id')::uuid,
    sqlc.arg('owner_user_id')::uuid,
    sqlc.narg('notebook_id')::uuid,
    sqlc.arg('client_id')::text,
    sqlc.arg('title')::text,
    sqlc.arg('blocks')::jsonb,
    sqlc.arg('body_text')::text,
    sqlc.arg('owner_user_id')::uuid
)
ON CONFLICT (family_id, owner_user_id, client_id) WHERE client_id <> ''
DO UPDATE SET updated_at = notes.updated_at
RETURNING
    id, family_id, owner_user_id, notebook_id, client_id, title, blocks, body_text,
    starred, archived, version, last_edited_by_user_id, created_at, updated_at;

-- name: GetNote :one
SELECT
    n.id, n.family_id, n.owner_user_id, n.notebook_id, n.client_id, n.title, n.blocks,
    n.body_text, n.starred, n.archived, n.version, n.last_edited_by_user_id, n.created_at,
    n.updated_at,
    -- `shared` is "can anybody but the owner reach this note", which is what the list
    -- row's glyph claims: a direct share, or a share of the notebook it sits in — that
    -- notebook being its owner's, by the rule in the header. A note sitting in someone
    -- ELSE's shared notebook is reachable by nobody, so it is not shared, and saying so
    -- would be the one lie the flag exists to prevent.
    (EXISTS (SELECT 1 FROM note_shares s WHERE s.note_id = n.id)
     OR EXISTS (SELECT 1 FROM notebook_shares ns
                 JOIN notebooks nbo ON nbo.id = ns.notebook_id
                 WHERE ns.notebook_id = n.notebook_id
                   AND nbo.owner_user_id = n.owner_user_id)
    )::bool AS shared,
    (SELECT count(*) FROM jsonb_array_elements(n.blocks) b
      WHERE b->>'type' IN ('3', 'BLOCK_TYPE_TODO'))::int AS task_total,
    (SELECT count(*) FROM jsonb_array_elements(n.blocks) b
      WHERE b->>'type' IN ('3', 'BLOCK_TYPE_TODO')
        AND coalesce((b->>'checked')::bool, false))::int AS task_done,
    left(n.body_text, 140) AS preview,
    (n.owner_user_id = sqlc.arg('user_id')::uuid
     OR EXISTS (SELECT 1 FROM note_shares s
                 WHERE s.note_id = n.id AND s.permission = 2
                   AND (s.subject = 2 OR s.member_user_id = sqlc.arg('user_id')::uuid))
     OR EXISTS (SELECT 1 FROM notebook_shares ns
                 JOIN notebooks nbo ON nbo.id = ns.notebook_id
                 WHERE ns.notebook_id = n.notebook_id AND ns.permission = 2
                   AND nbo.owner_user_id = n.owner_user_id
                   AND (ns.subject = 2 OR ns.member_user_id = sqlc.arg('user_id')::uuid))
    )::bool AS can_edit
FROM notes n
WHERE n.id = sqlc.arg('note_id')::uuid
  AND n.family_id = sqlc.arg('family_id')::uuid
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

-- GetNoteEditPermission answers "may this caller write this note?" in one round trip, so the
-- handler's write paths never rebuild the rule from owner_user_id and a share list. It
-- returns no row when the note is not even visible, which the handler reports as NotFound
-- rather than PermissionDenied — a note you cannot see must not confirm that it exists.
-- name: GetNoteEditPermission :one
SELECT
    n.id,
    n.owner_user_id,
    n.notebook_id,
    n.version,
    (n.owner_user_id = sqlc.arg('user_id')::uuid
     OR EXISTS (SELECT 1 FROM note_shares s
                 WHERE s.note_id = n.id AND s.permission = 2
                   AND (s.subject = 2 OR s.member_user_id = sqlc.arg('user_id')::uuid))
     OR EXISTS (SELECT 1 FROM notebook_shares ns
                 JOIN notebooks nbo ON nbo.id = ns.notebook_id
                 WHERE ns.notebook_id = n.notebook_id AND ns.permission = 2
                   AND nbo.owner_user_id = n.owner_user_id
                   AND (ns.subject = 2 OR ns.member_user_id = sqlc.arg('user_id')::uuid))
    )::bool AS can_edit
FROM notes n
WHERE n.id = sqlc.arg('note_id')::uuid
  AND n.family_id = sqlc.arg('family_id')::uuid
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

-- ListNotes is the list pane, the Archive rail row included. Every filter is a no-op sentinel
-- when unset (NULL for the notebook, false for the flags) so the app sends one request shape
-- whatever it is showing, and sorting is a closed text discriminator (NoteSort in the proto)
-- rather than SQL built in Go. page_size 0 means "no limit": NULLIF turns it into a NULL
-- LIMIT, which Postgres reads as unlimited.
-- name: ListNotes :many
SELECT
    n.id, n.family_id, n.owner_user_id, n.notebook_id, n.client_id, n.title, n.blocks,
    n.body_text, n.starred, n.archived, n.version, n.last_edited_by_user_id, n.created_at,
    n.updated_at,
    -- `shared` is "can anybody but the owner reach this note", which is what the list
    -- row's glyph claims: a direct share, or a share of the notebook it sits in — that
    -- notebook being its owner's, by the rule in the header. A note sitting in someone
    -- ELSE's shared notebook is reachable by nobody, so it is not shared, and saying so
    -- would be the one lie the flag exists to prevent.
    (EXISTS (SELECT 1 FROM note_shares s WHERE s.note_id = n.id)
     OR EXISTS (SELECT 1 FROM notebook_shares ns
                 JOIN notebooks nbo ON nbo.id = ns.notebook_id
                 WHERE ns.notebook_id = n.notebook_id
                   AND nbo.owner_user_id = n.owner_user_id)
    )::bool AS shared,
    (SELECT count(*) FROM jsonb_array_elements(n.blocks) b
      WHERE b->>'type' IN ('3', 'BLOCK_TYPE_TODO'))::int AS task_total,
    (SELECT count(*) FROM jsonb_array_elements(n.blocks) b
      WHERE b->>'type' IN ('3', 'BLOCK_TYPE_TODO')
        AND coalesce((b->>'checked')::bool, false))::int AS task_done,
    left(n.body_text, 140) AS preview,
    (n.owner_user_id = sqlc.arg('user_id')::uuid
     OR EXISTS (SELECT 1 FROM note_shares s
                 WHERE s.note_id = n.id AND s.permission = 2
                   AND (s.subject = 2 OR s.member_user_id = sqlc.arg('user_id')::uuid))
     OR EXISTS (SELECT 1 FROM notebook_shares ns
                 JOIN notebooks nbo ON nbo.id = ns.notebook_id
                 WHERE ns.notebook_id = n.notebook_id AND ns.permission = 2
                   AND nbo.owner_user_id = n.owner_user_id
                   AND (ns.subject = 2 OR ns.member_user_id = sqlc.arg('user_id')::uuid))
    )::bool AS can_edit
FROM notes n
WHERE n.family_id = sqlc.arg('family_id')::uuid
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
  AND (sqlc.narg('notebook_id')::uuid IS NULL OR n.notebook_id = sqlc.narg('notebook_id')::uuid)
  AND (NOT sqlc.arg('starred_only')::bool OR n.starred)
  -- archived_only implies include_archived: asking for the archive and having to remember to
  -- also ask for archived notes to be included would be a filter that means nothing on its own.
  AND (sqlc.arg('include_archived')::bool OR sqlc.arg('archived_only')::bool OR NOT n.archived)
  AND (NOT sqlc.arg('archived_only')::bool OR n.archived)
  AND (NOT sqlc.arg('shared_only')::bool OR n.owner_user_id <> sqlc.arg('user_id')::uuid)
ORDER BY
    CASE WHEN sqlc.arg('sort')::text = 'title' THEN lower(btrim(n.title)) END ASC,
    CASE WHEN sqlc.arg('sort')::text = 'created' THEN n.created_at END DESC,
    n.updated_at DESC
LIMIT NULLIF(sqlc.arg('page_size')::int, 0);

-- ListSharedWithMe is the sidebar's "Shared with me": visible, but not owned. It is
-- deliberately the same projection as ListNotes so the app renders one row component.
-- name: ListSharedWithMe :many
SELECT
    n.id, n.family_id, n.owner_user_id, n.notebook_id, n.client_id, n.title, n.blocks,
    n.body_text, n.starred, n.archived, n.version, n.last_edited_by_user_id, n.created_at,
    n.updated_at,
    -- `shared` is "can anybody but the owner reach this note", which is what the list
    -- row's glyph claims: a direct share, or a share of the notebook it sits in — that
    -- notebook being its owner's, by the rule in the header. A note sitting in someone
    -- ELSE's shared notebook is reachable by nobody, so it is not shared, and saying so
    -- would be the one lie the flag exists to prevent.
    (EXISTS (SELECT 1 FROM note_shares s WHERE s.note_id = n.id)
     OR EXISTS (SELECT 1 FROM notebook_shares ns
                 JOIN notebooks nbo ON nbo.id = ns.notebook_id
                 WHERE ns.notebook_id = n.notebook_id
                   AND nbo.owner_user_id = n.owner_user_id)
    )::bool AS shared,
    (SELECT count(*) FROM jsonb_array_elements(n.blocks) b
      WHERE b->>'type' IN ('3', 'BLOCK_TYPE_TODO'))::int AS task_total,
    (SELECT count(*) FROM jsonb_array_elements(n.blocks) b
      WHERE b->>'type' IN ('3', 'BLOCK_TYPE_TODO')
        AND coalesce((b->>'checked')::bool, false))::int AS task_done,
    left(n.body_text, 140) AS preview,
    (EXISTS (SELECT 1 FROM note_shares s
              WHERE s.note_id = n.id AND s.permission = 2
                AND (s.subject = 2 OR s.member_user_id = sqlc.arg('user_id')::uuid))
     OR EXISTS (SELECT 1 FROM notebook_shares ns
                 JOIN notebooks nbo ON nbo.id = ns.notebook_id
                 WHERE ns.notebook_id = n.notebook_id AND ns.permission = 2
                   AND nbo.owner_user_id = n.owner_user_id
                   AND (ns.subject = 2 OR ns.member_user_id = sqlc.arg('user_id')::uuid))
    )::bool AS can_edit
FROM notes n
WHERE n.family_id = sqlc.arg('family_id')::uuid
  AND n.owner_user_id <> sqlc.arg('user_id')::uuid
  AND NOT n.archived
  AND (
    EXISTS (SELECT 1 FROM note_shares s
             WHERE s.note_id = n.id
               AND (s.subject = 2 OR s.member_user_id = sqlc.arg('user_id')::uuid))
    OR EXISTS (SELECT 1 FROM notebook_shares ns
                JOIN notebooks nbo ON nbo.id = ns.notebook_id
                WHERE ns.notebook_id = n.notebook_id
                  AND nbo.owner_user_id = n.owner_user_id
                  AND (ns.subject = 2 OR ns.member_user_id = sqlc.arg('user_id')::uuid))
  )
ORDER BY n.updated_at DESC
LIMIT NULLIF(sqlc.arg('page_size')::int, 0);

-- UpdateNote writes the whole document and bumps version in the same statement. Two guards
-- sit in the WHERE clause, and both are there rather than in Go so that two concurrent
-- editors cannot both pass a check and then both write:
--
--   * the write predicate — the owner, or an EDIT share on the note or on its notebook (the
--     notebook disjunct, from the header: its owner's notes only). This is the statement that
--     used to be the exception; the file's rule is that a statement which forgets the
--     predicate must write nothing, and an exception makes the rule unreliable everywhere.
--   * expected_version: 0 forces the write, any other value must match the stored version.
--
-- Both refusals look identical from here — no row — and they are different rpc errors, so the
-- handler does not guess. It re-reads the permission on the empty path and answers
-- PermissionDenied, NotFound or ABORTED accordingly (updateRefusal in handler.go).
-- name: UpdateNote :one
UPDATE notes
SET title = sqlc.arg('title')::text,
    blocks = sqlc.arg('blocks')::jsonb,
    body_text = sqlc.arg('body_text')::text,
    version = version + 1,
    last_edited_by_user_id = sqlc.arg('user_id')::uuid,
    updated_at = NOW()
WHERE id = sqlc.arg('note_id')::uuid
  AND family_id = sqlc.arg('family_id')::uuid
  AND (
    owner_user_id = sqlc.arg('user_id')::uuid
    OR EXISTS (SELECT 1 FROM note_shares s
                WHERE s.note_id = notes.id AND s.permission = 2
                  AND (s.subject = 2 OR s.member_user_id = sqlc.arg('user_id')::uuid))
    OR EXISTS (SELECT 1 FROM notebook_shares ns
                JOIN notebooks nbo ON nbo.id = ns.notebook_id
                WHERE ns.notebook_id = notes.notebook_id AND ns.permission = 2
                  AND nbo.owner_user_id = notes.owner_user_id
                  AND (ns.subject = 2 OR ns.member_user_id = sqlc.arg('user_id')::uuid))
  )
  AND (sqlc.arg('expected_version')::bigint = 0
       OR version = sqlc.arg('expected_version')::bigint)
RETURNING
    id, family_id, owner_user_id, notebook_id, client_id, title, blocks, body_text,
    starred, archived, version, last_edited_by_user_id, created_at, updated_at;

-- Filing a note is the owner's alone, like deleting it: which notebook a note sits in decides
-- who else can read it, because the visibility predicate reaches a note through the notebook
-- its OWNER filed it in. Moving is therefore a re-share of one's own note, and an EDIT share
-- must not be a way to move someone else's note at all — not into a shared folder, which the
-- header's rule already makes inert, and not out of one, which would revoke a share the owner
-- granted. The owner check is here rather than only in the handler for the same reason every
-- other rule is: a statement that forgets it must write nothing.
-- name: MoveNote :one
UPDATE notes
SET notebook_id = sqlc.narg('notebook_id')::uuid,
    last_edited_by_user_id = sqlc.arg('user_id')::uuid,
    updated_at = NOW()
WHERE id = sqlc.arg('note_id')::uuid
  AND family_id = sqlc.arg('family_id')::uuid
  AND owner_user_id = sqlc.arg('user_id')::uuid
RETURNING
    id, family_id, owner_user_id, notebook_id, client_id, title, blocks, body_text,
    starred, archived, version, last_edited_by_user_id, created_at, updated_at;

-- Starring is one column on the shared row, not a per-viewer flag: it changes the star for
-- everyone who can see the note. That makes it a write, so it carries the write predicate —
-- the owner, or an EDIT share on the note or on its notebook — exactly as SetNoteArchived
-- does. The handler gates it too (mustEditNote), and this clause is not that check written
-- twice: it is the rule this file states about itself, that a statement which forgets it must
-- write nothing rather than trust a caller upstream to have remembered.
--
-- Starring does NOT bump version or last_edited_by_user_id: treating it as an edit would make
-- everyone else's next save conflict. It leaves updated_at alone as well, unlike archiving —
-- a star must not reorder the recently-updated list.
-- name: SetNoteStarred :one
UPDATE notes
SET starred = sqlc.arg('starred')::bool
WHERE id = sqlc.arg('note_id')::uuid
  AND family_id = sqlc.arg('family_id')::uuid
  AND (
    owner_user_id = sqlc.arg('user_id')::uuid
    OR EXISTS (SELECT 1 FROM note_shares s
                WHERE s.note_id = notes.id AND s.permission = 2
                  AND (s.subject = 2 OR s.member_user_id = sqlc.arg('user_id')::uuid))
    OR EXISTS (SELECT 1 FROM notebook_shares ns
                JOIN notebooks nbo ON nbo.id = ns.notebook_id
                WHERE ns.notebook_id = notes.notebook_id AND ns.permission = 2
                  AND nbo.owner_user_id = notes.owner_user_id
                  AND (ns.subject = 2 OR ns.member_user_id = sqlc.arg('user_id')::uuid))
  )
RETURNING
    id, family_id, owner_user_id, notebook_id, client_id, title, blocks, body_text,
    starred, archived, version, last_edited_by_user_id, created_at, updated_at;

-- Archiving takes the note out of everybody's list, not just the caller's, so it is a write
-- and carries the write predicate: the owner, or an EDIT share on the note or on its notebook.
-- The handler gates it too (mustEditNote), and this clause is not that check written twice —
-- it is the rule this file states about itself, that a statement which forgets it must write
-- nothing rather than trust a caller upstream to have remembered.
--
-- Like starring it leaves version and last_edited_by_user_id alone: archiving is filing, not
-- editing, and bumping the version would make everyone else's next save conflict. It does
-- touch updated_at, because the archive list is ordered by it.
-- name: SetNoteArchived :one
UPDATE notes
SET archived = sqlc.arg('archived')::bool,
    updated_at = NOW()
WHERE id = sqlc.arg('note_id')::uuid
  AND family_id = sqlc.arg('family_id')::uuid
  AND (
    owner_user_id = sqlc.arg('user_id')::uuid
    OR EXISTS (SELECT 1 FROM note_shares s
                WHERE s.note_id = notes.id AND s.permission = 2
                  AND (s.subject = 2 OR s.member_user_id = sqlc.arg('user_id')::uuid))
    OR EXISTS (SELECT 1 FROM notebook_shares ns
                JOIN notebooks nbo ON nbo.id = ns.notebook_id
                WHERE ns.notebook_id = notes.notebook_id AND ns.permission = 2
                  AND nbo.owner_user_id = notes.owner_user_id
                  AND (ns.subject = 2 OR ns.member_user_id = sqlc.arg('user_id')::uuid))
  )
RETURNING
    id, family_id, owner_user_id, notebook_id, client_id, title, blocks, body_text,
    starred, archived, version, last_edited_by_user_id, created_at, updated_at;

-- The owner alone can delete: a share, even an EDIT one, does not grant destruction. Shares,
-- comments and activity go with the row by ON DELETE CASCADE.
-- name: DeleteNote :execrows
DELETE FROM notes
WHERE id = sqlc.arg('note_id')::uuid
  AND family_id = sqlc.arg('family_id')::uuid
  AND owner_user_id = sqlc.arg('user_id')::uuid;
