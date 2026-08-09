-- Recipe photos live in MinIO (libs/go/storage); this column holds the public URL, not the
-- object bytes. Empty string means "no photo", matching description's NOT NULL DEFAULT ''
-- convention rather than adding a nullable column.
ALTER TABLE recipes ADD COLUMN IF NOT EXISTS image_url TEXT NOT NULL DEFAULT '';
