-- +goose Up
-- the page "/" opens: the overview of everything the user tracks, or one type's home page for
-- someone who only uses MediaHub for, say, music
ALTER TABLE users ADD COLUMN IF NOT EXISTS home_page TEXT NOT NULL DEFAULT 'overview'
	CHECK (home_page IN ('overview', 'anime', 'movie', 'manga', 'music', 'light_novel'));

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS home_page;
