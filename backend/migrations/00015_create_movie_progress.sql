-- +goose Up
CREATE TABLE user_movie_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    media_item_id UUID NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    position_secs FLOAT NOT NULL DEFAULT 0,
    duration_secs FLOAT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, media_item_id)
);

-- +goose Down
DROP TABLE user_movie_progress;
