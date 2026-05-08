-- +goose NO TRANSACTION

-- +goose Up
CREATE TYPE download_handler AS ENUM ('sonarr', 'radarr', 'mangal', 'ytdlp');

ALTER TABLE download_jobs
ADD COLUMN handler download_handler NOT NULL DEFAULT 'ytdlp';

CREATE TABLE sonarr_items (
    media_item_id UUID PRIMARY KEY REFERENCES media_items(id) ON DELETE CASCADE,
    sonarr_series_id INT NOT NULL,
    last_synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE radarr_items (
    media_item_id UUID PRIMARY KEY REFERENCES media_items(id) ON DELETE CASCADE,
    radarr_movie_id INT NOT NULL,
    last_synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sonarr_items_series ON sonarr_items(sonarr_series_id);
CREATE INDEX idx_radarr_items_movie ON radarr_items(radarr_movie_id);

-- +goose Down
DROP INDEX idx_radarr_items_movie;
DROP INDEX idx_sonarr_items_series;
DROP TABLE radarr_items;
DROP TABLE sonarr_items;
ALTER TABLE download_jobs DROP COLUMN handler;
DROP TYPE download_handler;