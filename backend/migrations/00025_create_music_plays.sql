-- +goose Up
-- one row each time someone listens to a track for long enough to count as a play
CREATE TABLE music_plays (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    media_item_id UUID NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    played_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_music_plays_played_at ON music_plays (played_at);

-- +goose Down
DROP TABLE music_plays;
