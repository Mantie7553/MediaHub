-- +goose Up
-- the TMDB movie a movie title is, so a movie deleted in Radarr and added there again finds
-- its old title instead of making a second one. Filled by the Radarr sync
ALTER TABLE media_items ADD COLUMN IF NOT EXISTS tmdb_id INT;
CREATE UNIQUE INDEX IF NOT EXISTS media_items_movie_tmdb_id
    ON media_items (tmdb_id) WHERE type = 'movie' AND tmdb_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS media_items_movie_tmdb_id;
ALTER TABLE media_items DROP COLUMN IF EXISTS tmdb_id;
