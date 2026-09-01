-- Comments and the activity rail. Both hang off a note by ON DELETE CASCADE, and both gate on
-- the note's read visibility, the predicate from notes.sql copied whole — notebook disjunct
-- included, so a notebook share opens the margins of its owner's notes and of no others. If
-- you can see the note you can read and add to its margin. Resolving is narrower — see
-- ResolveComment.

-- AddComment is an INSERT ... SELECT over the visible note, so a caller who cannot see the
-- note writes nothing and gets no row back, rather than the handler checking first and the
-- check drifting away from this file.
-- name: AddComment :one
INSERT INTO note_comments (note_id, author_user_id, body)
SELECT n.id, sqlc.arg('author_user_id')::uuid, sqlc.arg('body')::text
FROM notes n
WHERE n.id = sqlc.arg('note_id')::uuid
  AND n.family_id = sqlc.arg('family_id')::uuid
  AND (
    n.owner_user_id = sqlc.arg('author_user_id')::uuid
    OR EXISTS (SELECT 1 FROM note_shares s
                WHERE s.note_id = n.id
                  AND (s.subject = 2 OR s.member_user_id = sqlc.arg('author_user_id')::uuid))
    OR EXISTS (SELECT 1 FROM notebook_shares ns
                JOIN notebooks nbo ON nbo.id = ns.notebook_id
                WHERE ns.notebook_id = n.notebook_id
                  AND nbo.owner_user_id = n.owner_user_id
                  AND (ns.subject = 2 OR ns.member_user_id = sqlc.arg('author_user_id')::uuid))
  )
RETURNING *;

-- name: ListComments :many
SELECT c.*
FROM note_comments c
JOIN notes n ON n.id = c.note_id
WHERE c.note_id = sqlc.arg('note_id')::uuid
  AND n.family_id = sqlc.arg('family_id')::uuid
  AND (sqlc.arg('include_resolved')::bool OR NOT c.resolved)
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
ORDER BY c.created_at;

-- Resolving is not a read-level action: the comment's author or the note's owner may do it.
-- Anyone else who can see the note can reply to it, but cannot close someone else's thread.
--
-- Authority is ANDed with visibility, never substituted for it. The author clause outlives the
-- access that produced the comment: a share can be revoked after the thread was written, and
-- without the second half of this predicate the former sharee could still reach into the note
-- and flip a row in it. The note's owner always satisfies both halves, so the rule costs them
-- nothing.
-- name: ResolveComment :one
UPDATE note_comments c
SET resolved = sqlc.arg('resolved')::bool
FROM notes n
WHERE c.id = sqlc.arg('comment_id')::uuid
  AND n.id = c.note_id
  AND n.family_id = sqlc.arg('family_id')::uuid
  AND (c.author_user_id = sqlc.arg('user_id')::uuid
       OR n.owner_user_id = sqlc.arg('user_id')::uuid)
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
RETURNING c.*;

-- Activity is written by the handler, never by the app: the rows are a log of what the
-- service did, so there is no visibility predicate on the insert — the handler only reaches
-- here after the action it is recording already succeeded.
-- name: AddActivity :one
INSERT INTO note_activity (note_id, actor_user_id, kind, detail)
VALUES (
    sqlc.arg('note_id')::uuid,
    sqlc.arg('actor_user_id')::uuid,
    sqlc.arg('kind')::smallint,
    sqlc.arg('detail')::text
)
RETURNING *;

-- name: ListActivity :many
SELECT a.*
FROM note_activity a
JOIN notes n ON n.id = a.note_id
WHERE a.note_id = sqlc.arg('note_id')::uuid
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
  )
ORDER BY a.created_at DESC
LIMIT NULLIF(sqlc.arg('limit_count')::int, 0);

-- AddNotebookShareActivity records one share of a whole notebook on every note inside it.
-- The activity rail hangs off a note — note_activity has no notebook_id — so the choice is
-- either a row per note or no row at all, and "no row at all" is what made ACTIVITY_KIND_SHARED
-- unreachable. It is one set-based INSERT rather than a loop in the handler, so sharing a
-- notebook of a thousand notes is still one statement. Archived notes are skipped: their rail
-- is not on screen.
--
-- The join to notebooks is the notebook-share rule from notes.sql, not decoration: a notebook
-- share reaches only the notes its OWNER filed there. Without it, sharing a notebook stamps
-- "shared with X" onto the rail of every note inside it, including the ones belonging to other
-- members that the share does not reach -- telling them their private note was handed out, and
-- naming a person who cannot in fact read it.
-- name: AddNotebookShareActivity :exec
INSERT INTO note_activity (note_id, actor_user_id, kind, detail)
SELECT n.id,
       sqlc.arg('actor_user_id')::uuid,
       sqlc.arg('kind')::smallint,
       sqlc.arg('detail')::text
FROM notes n
JOIN notebooks nbo ON nbo.id = n.notebook_id
WHERE n.notebook_id = sqlc.arg('notebook_id')::uuid
  AND n.family_id = sqlc.arg('family_id')::uuid
  AND nbo.owner_user_id = n.owner_user_id
  AND NOT n.archived;
