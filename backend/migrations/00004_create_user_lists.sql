-- +goose Up
CREATE TABLE user_media_status (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    media_item_id UUID REFERENCES media_items(id) ON DELETE CASCADE,
    album_id UUID REFERENCES albums(id) ON DELETE CASCADE,
    status media_status NOT NULL,
    rating INT CHECK (rating BETWEEN 1 AND 5),
    notes TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, media_item_id),
    UNIQUE(user_id, album_id),
    CHECK (
        (media_item_id IS NOT NULL AND album_id IS NULL) OR
        (album_id IS NOT NULL AND media_item_id IS NULL)
    )
);

CREATE TABLE user_anime_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    media_item_id UUID NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    season_id UUID REFERENCES anime_seasons(id) ON DELETE CASCADE,
    episodes_watched INT NOT NULL DEFAULT 0,
    last_watched_at TIMESTAMPTZ,
    UNIQUE(user_id, media_item_id, season_id)
);

-- +goose Down
DROP TABLE user_anime_progress;
DROP TABLE user_media_status;