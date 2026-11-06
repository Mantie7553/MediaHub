-- +goose Up
-- links a manga to the Suwayomi source and title its missing chapters are downloaded from, picked
-- by an admin. Lookups go by these ids, never by title
CREATE TABLE suwayomi_items (
    media_item_id UUID PRIMARY KEY REFERENCES media_items(id) ON DELETE CASCADE,
    source_id TEXT NOT NULL,
    source_name TEXT NOT NULL,
    suwayomi_manga_id INT NOT NULL,
    title TEXT NOT NULL,
    linked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE suwayomi_items;
