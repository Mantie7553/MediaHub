-- +goose Up
-- the days each user was around, for the Insights activity chart
CREATE TABLE user_active_days (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    day DATE NOT NULL,
    PRIMARY KEY (user_id, day)
);
CREATE INDEX idx_user_active_days_day ON user_active_days (day);

-- what history there is: the last time each title's progress was saved
INSERT INTO user_active_days (user_id, day)
SELECT DISTINCT user_id, updated_at::date FROM (
    SELECT user_id, updated_at FROM user_episode_progress
    UNION ALL SELECT user_id, updated_at FROM user_movie_progress
    UNION ALL SELECT user_id, updated_at FROM manga_progress
    UNION ALL SELECT user_id, updated_at FROM light_novel_progress
    UNION ALL SELECT id, last_seen_at FROM users
) a
WHERE user_id IS NOT NULL AND updated_at IS NOT NULL
ON CONFLICT DO NOTHING;

-- the library's size once a day, for the Insights storage chart
CREATE TABLE storage_snapshots (
    day DATE PRIMARY KEY,
    library_bytes BIGINT NOT NULL,
    -- bytes per media type, {"anime": 123, ...}
    folders JSONB NOT NULL DEFAULT '{}',
    disk_used BIGINT,
    disk_total BIGINT
);

-- +goose Down
DROP TABLE storage_snapshots;
DROP TABLE user_active_days;
