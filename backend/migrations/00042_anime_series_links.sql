-- +goose Up
-- AniList entries an admin placed in a TVDB series by hand, for the ones Fribb's anime list
-- leaves out of their series (or puts in the wrong one). Laid over Fribb's list everywhere
CREATE TABLE IF NOT EXISTS anime_series_links (
    anilist_id INT PRIMARY KEY,
    tvdb_id INT NOT NULL,
    season_number INT NOT NULL CHECK (season_number >= 0),
    -- the name of the title it was attached from, to list the link by once that title is merged
    title TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS anime_series_links;
