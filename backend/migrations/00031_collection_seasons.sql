-- +goose Up
-- an anime entry in a collection can be one of its seasons rather than the whole show, so a
-- collection can be a watch order (season 1, a movie, season 2, ...). A collection holds either
-- the whole anime or some of its seasons, which the handler checks
ALTER TABLE collection_items ADD COLUMN season_number INT
    CHECK (season_number IS NULL OR media_item_id IS NOT NULL);

DROP INDEX uq_collection_media;
CREATE UNIQUE INDEX uq_collection_media ON collection_items(collection_id, media_item_id)
    WHERE media_item_id IS NOT NULL AND season_number IS NULL;
CREATE UNIQUE INDEX uq_collection_season ON collection_items(collection_id, media_item_id, season_number)
    WHERE season_number IS NOT NULL;

-- +goose Down
DELETE FROM collection_items WHERE season_number IS NOT NULL;
DROP INDEX uq_collection_season;
DROP INDEX uq_collection_media;
CREATE UNIQUE INDEX uq_collection_media ON collection_items(collection_id, media_item_id) WHERE media_item_id IS NOT NULL;
ALTER TABLE collection_items DROP COLUMN season_number;
