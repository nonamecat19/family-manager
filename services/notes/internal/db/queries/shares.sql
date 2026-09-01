-- Sharing. Only the OWNER of a note or notebook may grant or revoke a share, and that check
-- is in the statement rather than in Go: each write is an INSERT ... SELECT over the owned
-- row, so a non-owner's call writes nothing and returns no row instead of relying on the
-- handler having remembered to look first.
--
-- MEMBER and FAMILY are separate statements because they conflict on different partial unique
-- indexes, and one INSERT can name only one arbiter. Re-sharing to the same subject therefore
-- changes the permission on the row that exists instead of stacking a second grant.

-- name: ShareNoteWithMember :one
INSERT INTO note_shares (
    note_id, family_id, subject, member_user_id, permission, granted_by_user_id
)
SELECT n.id, n.family_id, 1,
       sqlc.arg('member_user_id')::uuid,
       sqlc.arg('permission')::smallint,
       sqlc.arg('granted_by_user_id')::uuid
FROM notes n
WHERE n.id = sqlc.arg('note_id')::uuid
  AND n.family_id = sqlc.arg('family_id')::uuid
  AND n.owner_user_id = sqlc.arg('granted_by_user_id')::uuid
ON CONFLICT (note_id, member_user_id) WHERE subject = 1
DO UPDATE SET permission = EXCLUDED.permission,
              granted_by_user_id = EXCLUDED.granted_by_user_id
RETURNING *;

-- name: ShareNoteWithFamily :one
INSERT INTO note_shares (
    note_id, family_id, subject, member_user_id, permission, granted_by_user_id
)
SELECT n.id, n.family_id, 2, NULL,
       sqlc.arg('permission')::smallint,
       sqlc.arg('granted_by_user_id')::uuid
FROM notes n
WHERE n.id = sqlc.arg('note_id')::uuid
  AND n.family_id = sqlc.arg('family_id')::uuid
  AND n.owner_user_id = sqlc.arg('granted_by_user_id')::uuid
ON CONFLICT (note_id) WHERE subject = 2
DO UPDATE SET permission = EXCLUDED.permission,
              granted_by_user_id = EXCLUDED.granted_by_user_id
RETURNING *;

-- name: ShareNotebookWithMember :one
INSERT INTO notebook_shares (
    notebook_id, family_id, subject, member_user_id, permission, granted_by_user_id
)
SELECT nb.id, nb.family_id, 1,
       sqlc.arg('member_user_id')::uuid,
       sqlc.arg('permission')::smallint,
       sqlc.arg('granted_by_user_id')::uuid
FROM notebooks nb
WHERE nb.id = sqlc.arg('notebook_id')::uuid
  AND nb.family_id = sqlc.arg('family_id')::uuid
  AND nb.owner_user_id = sqlc.arg('granted_by_user_id')::uuid
ON CONFLICT (notebook_id, member_user_id) WHERE subject = 1
DO UPDATE SET permission = EXCLUDED.permission,
              granted_by_user_id = EXCLUDED.granted_by_user_id
RETURNING *;

-- name: ShareNotebookWithFamily :one
INSERT INTO notebook_shares (
    notebook_id, family_id, subject, member_user_id, permission, granted_by_user_id
)
SELECT nb.id, nb.family_id, 2, NULL,
       sqlc.arg('permission')::smallint,
       sqlc.arg('granted_by_user_id')::uuid
FROM notebooks nb
WHERE nb.id = sqlc.arg('notebook_id')::uuid
  AND nb.family_id = sqlc.arg('family_id')::uuid
  AND nb.owner_user_id = sqlc.arg('granted_by_user_id')::uuid
ON CONFLICT (notebook_id) WHERE subject = 2
DO UPDATE SET permission = EXCLUDED.permission,
              granted_by_user_id = EXCLUDED.granted_by_user_id
RETURNING *;

-- name: DeleteNoteShare :execrows
DELETE FROM note_shares s
WHERE s.id = sqlc.arg('share_id')::uuid
  AND s.note_id = sqlc.arg('note_id')::uuid
  AND s.family_id = sqlc.arg('family_id')::uuid
  AND EXISTS (SELECT 1 FROM notes n
               WHERE n.id = s.note_id
                 AND n.owner_user_id = sqlc.arg('user_id')::uuid);

-- name: DeleteNotebookShare :execrows
DELETE FROM notebook_shares s
WHERE s.id = sqlc.arg('share_id')::uuid
  AND s.notebook_id = sqlc.arg('notebook_id')::uuid
  AND s.family_id = sqlc.arg('family_id')::uuid
  AND EXISTS (SELECT 1 FROM notebooks nb
               WHERE nb.id = s.notebook_id
                 AND nb.owner_user_id = sqlc.arg('user_id')::uuid);

-- Anyone who can see the note can see who else it is shared with — that is the point of the
-- avatar row in the header. The visibility predicate is therefore the read one from notes.sql,
-- not the ownership one, and it carries that file's notebook disjunct.
--
-- What comes back is the DIRECT shares only: the note_shares rows, not the notebook share the
-- note may also be reachable through. That is deliberate and it is what Unshare needs — these
-- are the rows a share_id can revoke, and listing a notebook_shares row here would offer the
-- app a revoke it cannot perform from a note (revoking it belongs to the notebook, where
-- ListNotebookShares shows it). The list-row `shared` flag in notes.sql is the wider fact —
-- "somebody other than the owner can reach this" — and it does count the notebook, so the two
-- disagreeing is the intended division, not a bug: one is reachability, this one is revocability.
-- name: ListNoteShares :many
SELECT s.*
FROM note_shares s
JOIN notes n ON n.id = s.note_id
WHERE s.note_id = sqlc.arg('note_id')::uuid
  AND n.family_id = sqlc.arg('family_id')::uuid
  AND (
    n.owner_user_id = sqlc.arg('user_id')::uuid
    OR EXISTS (SELECT 1 FROM note_shares s2
                WHERE s2.note_id = n.id
                  AND (s2.subject = 2 OR s2.member_user_id = sqlc.arg('user_id')::uuid))
    OR EXISTS (SELECT 1 FROM notebook_shares ns
                JOIN notebooks nbo ON nbo.id = ns.notebook_id
                WHERE ns.notebook_id = n.notebook_id
                  AND nbo.owner_user_id = n.owner_user_id
                  AND (ns.subject = 2 OR ns.member_user_id = sqlc.arg('user_id')::uuid))
  )
ORDER BY s.created_at;

-- name: ListNotebookShares :many
SELECT s.*
FROM notebook_shares s
JOIN notebooks nb ON nb.id = s.notebook_id
WHERE s.notebook_id = sqlc.arg('notebook_id')::uuid
  AND nb.family_id = sqlc.arg('family_id')::uuid
  AND (
    nb.owner_user_id = sqlc.arg('user_id')::uuid
    OR EXISTS (SELECT 1 FROM notebook_shares s2
                WHERE s2.notebook_id = nb.id
                  AND (s2.subject = 2 OR s2.member_user_id = sqlc.arg('user_id')::uuid))
  )
ORDER BY s.created_at;
