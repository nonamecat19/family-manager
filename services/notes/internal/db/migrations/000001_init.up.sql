-- services/notes owns Commonplace. Every row is scoped by family_id, which comes from the
-- access token's family_id claim (stamped by services/auth from services/family) — there is
-- NO foreign key to the family service's tables, by design: crossing a service boundary in
-- SQL is what the contract in libs/proto exists to prevent.
--
-- user_id columns likewise reference users in services/auth and carry no FK.
--
-- The unit of privacy here is the NOTE, not the family. A note belongs to owner_user_id and
-- is invisible to everyone else until a row in note_shares or notebook_shares says otherwise.
-- That rule lives in the queries (internal/db/queries/), not in the handler; the schema's job
-- is to make the predicate cheap — hence the share indexes below.

CREATE TABLE IF NOT EXISTS notebooks (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id      UUID NOT NULL,
    owner_user_id  UUID NOT NULL,
    -- The tree is one level deep in the design ("Product / Roadmap") but the column allows
    -- more, because forbidding it later is a migration and allowing it later is a contract
    -- change. SET NULL rather than CASCADE: deleting a parent must not silently take a
    -- subtree of notebooks (and the notes inside them) with it.
    parent_id      UUID REFERENCES notebooks (id) ON DELETE SET NULL,
    name           TEXT NOT NULL CHECK (length(btrim(name)) > 0),
    archived       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notebooks_family_owner
    ON notebooks (family_id, owner_user_id, name);
CREATE INDEX IF NOT EXISTS idx_notebooks_parent
    ON notebooks (parent_id);

CREATE TABLE IF NOT EXISTS notes (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id               UUID NOT NULL,
    owner_user_id           UUID NOT NULL,
    -- NULL means the note lives outside any notebook, which is where quick capture puts one.
    -- SET NULL on notebook delete for the same reason: a notebook is a folder, not an owner.
    notebook_id             UUID REFERENCES notebooks (id) ON DELETE SET NULL,
    -- client_id is the id the app minted while offline; '' means "no idempotency key". It is
    -- the arbiter of the partial unique index below, which is what makes CreateNote safe for
    -- the mobile capture queue to retry after a timeout it never saw the answer to.
    client_id               TEXT NOT NULL DEFAULT '',
    title                   TEXT NOT NULL DEFAULT '',
    -- blocks is the note's document: an ordered JSON array of Block (see the proto). It is a
    -- document rather than rows because it is always read and written whole.
    blocks                  JSONB NOT NULL DEFAULT '[]'::JSONB,
    -- body_text is the blocks' text flattened by the handler on every write. It exists so
    -- that search and the list-row preview never have to walk the JSON at read time.
    body_text               TEXT NOT NULL DEFAULT '',
    -- 'simple' rather than 'english': notes are written in whatever language the family
    -- speaks, and a stemmer for the wrong one is worse than none.
    search                  TSVECTOR GENERATED ALWAYS AS (
                                to_tsvector('simple',
                                    coalesce(title, '') || ' ' || coalesce(body_text, ''))
                            ) STORED,
    starred                 BOOLEAN NOT NULL DEFAULT FALSE,
    archived                BOOLEAN NOT NULL DEFAULT FALSE,
    -- version increments on every successful UpdateNote and backs the conditional write
    -- (UpdateNoteRequest.expected_version).
    version                 BIGINT NOT NULL DEFAULT 1,
    last_edited_by_user_id  UUID NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notes_search
    ON notes USING GIN (search);
-- The owner's own list, newest first — the common read, and the one the visibility predicate
-- short-circuits on for a note nobody has shared.
CREATE INDEX IF NOT EXISTS idx_notes_family_owner_updated
    ON notes (family_id, owner_user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_notes_notebook_updated
    ON notes (notebook_id, updated_at DESC);
-- Idempotent capture: one note per (family, owner, client_id), and only when the app supplied
-- one. Rows with client_id = '' are excluded, so notes created online do not collide.
CREATE UNIQUE INDEX IF NOT EXISTS idx_notes_client_id
    ON notes (family_id, owner_user_id, client_id)
    WHERE client_id <> '';

-- Share rows. subject and permission are the proto's ShareSubject / SharePermission enum
-- numbers (1 = MEMBER / VIEW, 2 = FAMILY / EDIT) stored as smallint rather than a text enum,
-- so the visibility predicate compares integers and the contract stays the single definition.
CREATE TABLE IF NOT EXISTS note_shares (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    note_id             UUID NOT NULL REFERENCES notes (id) ON DELETE CASCADE,
    family_id           UUID NOT NULL,
    subject             SMALLINT NOT NULL CHECK (subject IN (1, 2)),
    -- Set when subject = MEMBER, NULL when subject = FAMILY.
    member_user_id      UUID,
    permission          SMALLINT NOT NULL CHECK (permission IN (1, 2)),
    granted_by_user_id  UUID NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK ((subject = 1 AND member_user_id IS NOT NULL)
        OR (subject = 2 AND member_user_id IS NULL))
);

CREATE INDEX IF NOT EXISTS idx_note_shares_note
    ON note_shares (note_id);
-- The reverse lookup the visibility predicate and "Shared with me" both make.
CREATE INDEX IF NOT EXISTS idx_note_shares_member
    ON note_shares (member_user_id, note_id);
-- One row per (note, member) and one whole-family row per note: re-sharing changes the
-- permission on the row that is already there instead of stacking a second grant.
CREATE UNIQUE INDEX IF NOT EXISTS idx_note_shares_member_unique
    ON note_shares (note_id, member_user_id) WHERE subject = 1;
CREATE UNIQUE INDEX IF NOT EXISTS idx_note_shares_family_unique
    ON note_shares (note_id) WHERE subject = 2;

CREATE TABLE IF NOT EXISTS notebook_shares (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notebook_id         UUID NOT NULL REFERENCES notebooks (id) ON DELETE CASCADE,
    family_id           UUID NOT NULL,
    subject             SMALLINT NOT NULL CHECK (subject IN (1, 2)),
    member_user_id      UUID,
    permission          SMALLINT NOT NULL CHECK (permission IN (1, 2)),
    granted_by_user_id  UUID NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK ((subject = 1 AND member_user_id IS NOT NULL)
        OR (subject = 2 AND member_user_id IS NULL))
);

CREATE INDEX IF NOT EXISTS idx_notebook_shares_notebook
    ON notebook_shares (notebook_id);
CREATE INDEX IF NOT EXISTS idx_notebook_shares_member
    ON notebook_shares (member_user_id, notebook_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_notebook_shares_member_unique
    ON notebook_shares (notebook_id, member_user_id) WHERE subject = 1;
CREATE UNIQUE INDEX IF NOT EXISTS idx_notebook_shares_family_unique
    ON notebook_shares (notebook_id) WHERE subject = 2;

CREATE TABLE IF NOT EXISTS note_comments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    note_id         UUID NOT NULL REFERENCES notes (id) ON DELETE CASCADE,
    author_user_id  UUID NOT NULL,
    body            TEXT NOT NULL CHECK (length(btrim(body)) > 0),
    resolved        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_note_comments_note
    ON note_comments (note_id, created_at);

-- kind is the proto's ActivityKind enum number; detail is the object of the action (the task
-- text that was checked, the name a note was shared with), '' when the kind needs none.
CREATE TABLE IF NOT EXISTS note_activity (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    note_id        UUID NOT NULL REFERENCES notes (id) ON DELETE CASCADE,
    actor_user_id  UUID NOT NULL,
    kind           SMALLINT NOT NULL,
    detail         TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_note_activity_note
    ON note_activity (note_id, created_at DESC);
