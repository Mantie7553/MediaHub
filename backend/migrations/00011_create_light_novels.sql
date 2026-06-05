-- +goose NO TRANSACTION
-- +goose Up
ALTER TYPE media_type ADD VALUE 'light_novel';

CREATE TABLE light_novel_metadata (
    media_item_id UUID PRIMARY KEY NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    author TEXT,
    total_volumes INT,
    genres TEXT[]
);

CREATE TABLE light_novel_volumes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    media_item_id UUID NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    volume_number INT,
    title TEXT,
    file_path TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (media_item_id, volume_number)
);

CREATE TABLE light_novel_progress (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    volume_id UUID NOT NULL REFERENCES light_novel_volumes(id) ON DELETE CASCADE,
    media_item_id UUID NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    scroll_position FLOAT NOT NULL DEFAULT 0,
    completed BOOL NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, volume_id)
);

-- +goose Down
DROP TABLE light_novel_progress CASCADE;
DROP TABLE light_novel_volumes CASCADE;
DROP TABLE light_novel_metadata CASCADE;