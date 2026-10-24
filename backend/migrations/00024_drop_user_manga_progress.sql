-- +goose Up
-- never used: manga progress is kept per chapter in manga_progress
DROP TABLE user_manga_progress;

-- +goose Down
CREATE TABLE user_manga_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    media_item_id UUID NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    chapters_read INT NOT NULL DEFAULT 0,
    last_read_at TIMESTAMPTZ,
    UNIQUE(user_id, media_item_id)
);
