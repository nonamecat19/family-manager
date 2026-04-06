-- +goose Up
CREATE TABLE schema_info (
    version INT PRIMARY KEY DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO schema_info (version) VALUES (1);

-- +goose Down
DROP TABLE IF EXISTS schema_info;
