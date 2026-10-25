-- +goose Up
-- seconds of video each user actually played per title and day, for Insights' hours watched.
-- Saved positions only say how far someone got, not how long they watched
CREATE TABLE watch_time (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    media_item_id UUID NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    day DATE NOT NULL,
    seconds DOUBLE PRECISION NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, media_item_id, day)
);
CREATE INDEX idx_watch_time_day ON watch_time (day);

-- +goose Down
DROP TABLE watch_time;
