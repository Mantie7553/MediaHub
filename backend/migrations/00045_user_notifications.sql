-- +goose Up
-- users are told about their own requests and about titles they follow
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_kind_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_kind_check CHECK (kind IN (
    'approval_needed', 'download_failed', 'downloading', 'download_by_hand',
    'request_approved', 'request_rejected', 'title_ready', 'new_parts'
));

-- the kinds a user turned off: no row means they are told
CREATE TABLE IF NOT EXISTS notification_mutes (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    PRIMARY KEY (user_id, kind)
);

-- +goose Down
DROP TABLE IF EXISTS notification_mutes;
DELETE FROM notifications WHERE kind IN ('request_approved', 'request_rejected', 'title_ready', 'new_parts');
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_kind_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_kind_check CHECK (kind IN (
    'approval_needed', 'download_failed', 'downloading', 'download_by_hand'
));
