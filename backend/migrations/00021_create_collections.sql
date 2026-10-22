-- +goose Up
-- a user's playlists (tracks, in the order they arrange) and collections (titles, albums, tracks
-- and playlists grouped together, like a series). Only the owner sees one for now; sharing adds
-- visibility values and a members table, and the handlers check access in one place
CREATE TYPE collection_kind AS ENUM ('playlist', 'collection');
CREATE TYPE collection_visibility AS ENUM ('private');

CREATE TABLE collections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind collection_kind NOT NULL,
    name TEXT NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 100),
    description TEXT,
    visibility collection_visibility NOT NULL DEFAULT 'private',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_collections_owner ON collections(owner_id, kind);

-- exactly one of media_item_id (a track or title), album_id or child_collection_id (a playlist
-- inside a collection). Deleting any of them, or the collection, takes the entry with it
CREATE TABLE collection_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    collection_id UUID NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    media_item_id UUID REFERENCES media_items(id) ON DELETE CASCADE,
    album_id UUID REFERENCES albums(id) ON DELETE CASCADE,
    child_collection_id UUID REFERENCES collections(id) ON DELETE CASCADE,
    -- spaced apart so moving an entry between two others rarely renumbers the rest
    position BIGINT NOT NULL,
    -- who added it, for when other people can edit a shared playlist
    added_by UUID REFERENCES users(id) ON DELETE SET NULL,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (num_nonnulls(media_item_id, album_id, child_collection_id) = 1)
);
CREATE INDEX idx_collection_items_order ON collection_items(collection_id, position);
CREATE UNIQUE INDEX uq_collection_media ON collection_items(collection_id, media_item_id) WHERE media_item_id IS NOT NULL;
CREATE UNIQUE INDEX uq_collection_album ON collection_items(collection_id, album_id) WHERE album_id IS NOT NULL;
CREATE UNIQUE INDEX uq_collection_child ON collection_items(collection_id, child_collection_id) WHERE child_collection_id IS NOT NULL;
-- finding the collections an item is in, and cleaning up after deletes
CREATE INDEX idx_collection_items_media ON collection_items(media_item_id) WHERE media_item_id IS NOT NULL;
CREATE INDEX idx_collection_items_album ON collection_items(album_id) WHERE album_id IS NOT NULL;
CREATE INDEX idx_collection_items_child ON collection_items(child_collection_id) WHERE child_collection_id IS NOT NULL;

-- +goose Down
DROP TABLE collection_items;
DROP TABLE collections;
DROP TYPE collection_visibility;
DROP TYPE collection_kind;
