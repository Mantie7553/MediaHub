-- +goose Up
-- what someone is told about: one row per recipient, so any user can get them, though only
-- admins do for now. read_at is NULL until it's read
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('approval_needed', 'download_failed', 'downloading', 'download_by_hand')),
    title TEXT NOT NULL,
    body TEXT,
    -- the page to open, where the recipient acts on it
    link TEXT,
    -- what it is about ("request:<id>", "job:<id>", "downloading:<media item id>"): a recipient
    -- is told once, until they read it
    dedupe_key TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    read_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS notifications_unread_once
    ON notifications (user_id, dedupe_key) WHERE read_at IS NULL;
CREATE INDEX IF NOT EXISTS notifications_user_order
    ON notifications (user_id, (read_at IS NULL) DESC, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS notifications;
