-- +goose Up
-- when a user last signed in or refreshed their session, for the admin Insights page
ALTER TABLE users ADD COLUMN last_seen_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE users DROP COLUMN last_seen_at;
