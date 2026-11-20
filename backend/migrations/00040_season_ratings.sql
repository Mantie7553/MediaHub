-- +goose Up
-- a user's ratings of an anime series' seasons (TVDB seasons, as Sonarr and the season tabs keep
-- them). anilist_id 0 is a rating given to the season itself; others came from saving one
-- AniList/MyAnimeList entry of it, and are averaged when the season has no rating of its own
CREATE TABLE IF NOT EXISTS user_season_ratings (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    media_item_id UUID NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    season_number INT NOT NULL,
    anilist_id INT NOT NULL DEFAULT 0,
    rating INT NOT NULL CHECK (rating BETWEEN 1 AND 10),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, media_item_id, season_number, anilist_id)
);

-- false when the title's rating is the average of its season ratings rather than one the user
-- picked. A missing rating is filled from the seasons too
ALTER TABLE user_media_status ADD COLUMN IF NOT EXISTS rating_is_set BOOLEAN NOT NULL DEFAULT true;

-- +goose Down
ALTER TABLE user_media_status DROP COLUMN IF EXISTS rating_is_set;
DROP TABLE IF EXISTS user_season_ratings;
