-- +goose Up
-- the TVDB series an anime title is. AniList and MyAnimeList keep each season (Bakemonogatari,
-- Nisemonogatari, ...) as its own entry while Sonarr keeps the whole series, so this is what
-- every season's save and the Sonarr sync find the one title by
ALTER TABLE media_items ADD COLUMN IF NOT EXISTS tvdb_id INT;
CREATE UNIQUE INDEX IF NOT EXISTS media_items_anime_tvdb_id
    ON media_items (tvdb_id) WHERE type = 'anime' AND tvdb_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS media_items_anime_tvdb_id;
ALTER TABLE media_items DROP COLUMN IF EXISTS tvdb_id;
