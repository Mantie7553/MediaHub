-- +goose Up
CREATE TABLE user_episode_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    episode_id UUID NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
    position_secs FLOAT NOT NULL DEFAULT 0,
    duration_secs FLOAT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, episode_id)
);

-- +goose Down
DROP TABLE user_episode_progress;