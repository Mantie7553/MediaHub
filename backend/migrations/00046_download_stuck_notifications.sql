-- +goose Up
-- admins are told about Sonarr / Radarr downloads that are stuck or never found a release
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_kind_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_kind_check CHECK (kind IN (
    'approval_needed', 'download_failed', 'downloading', 'download_by_hand', 'download_stuck',
    'request_approved', 'request_rejected', 'title_ready', 'new_parts'
));

-- +goose Down
DELETE FROM notifications WHERE kind = 'download_stuck';
DELETE FROM notification_mutes WHERE kind = 'download_stuck';
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_kind_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_kind_check CHECK (kind IN (
    'approval_needed', 'download_failed', 'downloading', 'download_by_hand',
    'request_approved', 'request_rejected', 'title_ready', 'new_parts'
));
