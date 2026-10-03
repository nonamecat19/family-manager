ALTER TABLE notes ADD COLUMN owner_user_id UUID;
ALTER TABLE notes ADD COLUMN starred BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE notes ADD COLUMN archived BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE notes ADD COLUMN blocks TEXT;

CREATE INDEX idx_notes_starred ON notes (family_id, starred);
CREATE INDEX idx_notes_archived ON notes (family_id, archived);
