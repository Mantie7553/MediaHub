-- +goose Up
-- the MyAnimeList entry each Sonarr season is, from Fribb's anime list, for AniSkip lookups.
-- Seasons Fribb splits over several entries (split cours, specials) have no row
CREATE TABLE anime_season_mal (
    media_item_id UUID NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    season_number INT NOT NULL,
    mal_id INT NOT NULL,
    PRIMARY KEY (media_item_id, season_number)
);

-- AniSkip's openings/endings for an episode, so playing it again doesn't ask again.
-- file_path is the file they were fetched for, a new file (different cut) asks again
CREATE TABLE skip_times (
    episode_id UUID PRIMARY KEY REFERENCES episodes(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    segments JSONB NOT NULL,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE users ADD COLUMN auto_skip BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE users DROP COLUMN auto_skip;
DROP TABLE skip_times;
DROP TABLE anime_season_mal;
