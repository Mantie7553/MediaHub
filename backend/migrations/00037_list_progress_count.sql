-- +goose Up
-- how many episodes, chapters or volumes the user says they're through, for progress past the
-- files the library has (read elsewhere, not downloaded yet). NULL when the files say it all
ALTER TABLE user_media_status ADD COLUMN IF NOT EXISTS progress_count INT CHECK (progress_count >= 0);

-- +goose Down
ALTER TABLE user_media_status DROP COLUMN IF EXISTS progress_count;
