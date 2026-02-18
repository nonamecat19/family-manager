CREATE TABLE list_tags (
    list_id UUID NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (list_id, tag_id)
);

CREATE INDEX idx_list_tags_tag_id ON list_tags(tag_id);
