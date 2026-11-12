-- +goose Up
-- covers from other sites are downloaded and kept in MEDIA_ROOT/Covers. This is the address a
-- saved copy came from, so a sync handing back the same address keeps the copy instead of
-- pointing the title at the other site again
ALTER TABLE media_items ADD COLUMN cover_origin_url TEXT;

-- +goose Down
ALTER TABLE media_items DROP COLUMN cover_origin_url;
