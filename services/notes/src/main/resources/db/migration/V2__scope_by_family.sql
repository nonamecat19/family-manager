ALTER TABLE "groups" ADD COLUMN family_id UUID;
ALTER TABLE notes ADD COLUMN family_id UUID;

UPDATE "groups" SET family_id = '00000000-0000-0000-0000-000000000000' WHERE family_id IS NULL;
UPDATE notes SET family_id = '00000000-0000-0000-0000-000000000000' WHERE family_id IS NULL;

ALTER TABLE "groups" ALTER COLUMN family_id SET NOT NULL;
ALTER TABLE notes ALTER COLUMN family_id SET NOT NULL;

CREATE INDEX idx_groups_family_id ON "groups" (family_id);
CREATE INDEX idx_notes_family_id ON notes (family_id, updated_at DESC);

DROP INDEX idx_notes_group_id;
DROP INDEX idx_notes_status;
CREATE INDEX idx_notes_group_id ON notes (family_id, group_id);
CREATE INDEX idx_notes_status ON notes (family_id, status);
