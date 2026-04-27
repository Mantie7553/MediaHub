-- +goose Up
CREATE TABLE media_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type media_type NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    cover_image_url TEXT,
    release_date DATE,
    external_id TEXT,
    external_source external_source,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE anime_metadata (
    media_item_id UUID PRIMARY KEY REFERENCES media_items(id) ON DELETE CASCADE,
    studio TEXT,
    status anime_status,
    genres TEXT[]
);

CREATE TABLE anime_seasons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    media_item_id UUID NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    season_number INT NOT NULL,
    episode_count INT NOT NULL,
    title TEXT,
    air_date DATE,
    UNIQUE(media_item_id, season_number)
);

CREATE TABLE movie_metadata (
    media_item_id UUID PRIMARY KEY REFERENCES media_items(id) ON DELETE CASCADE,
    runtime_mins INT,
    director TEXT,
    genres TEXT[]
);

CREATE TABLE albums (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    artist TEXT NOT NULL,
    release_date DATE,
    cover_image_url TEXT,
    external_id TEXT,
    external_source external_source,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE music_metadata (
    media_item_id UUID PRIMARY KEY REFERENCES media_items(id) ON DELETE CASCADE,
    artist TEXT NOT NULL,
    album_id UUID REFERENCES albums(id) ON DELETE SET NULL,
    track_number INT,
    duration_secs INT,
    genres TEXT[]
);

-- +goose Down
DROP TABLE music_metadata;
DROP TABLE albums;
DROP TABLE movie_metadata;
DROP TABLE anime_seasons;
DROP TABLE anime_metadata;
DROP TABLE media_items;