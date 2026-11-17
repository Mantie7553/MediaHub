-- +goose Up
-- when the library sync last enriched a title from AniList, so a nightly sync only asks again
-- once that is a week old. NULL: never, asked on the next sync
ALTER TABLE media_items ADD COLUMN IF NOT EXISTS metadata_synced_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE media_items DROP COLUMN IF EXISTS metadata_synced_at;
