-- +goose Up
CREATE TABLE plex_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    media_item_id UUID NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    plex_rating_key TEXT NOT NULL,
    plex_library_id TEXT NOT NULL,
    plex_metadata_url TEXT,
    last_synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_library_access (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    media_item_id UUID NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE(user_id, media_item_id)
);

-- +goose Down
DROP TABLE user_library_access;
DROP TABLE plex_items;