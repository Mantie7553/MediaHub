-- +goose Up
-- an image an admin uploaded for a playlist or collection, shown instead of the mosaic of what's
-- in it. The file is MEDIA_ROOT/Covers/<image_id>; copies (featured ones, saved ones) share the
-- same image, and the file goes once no collection uses it
ALTER TABLE collections ADD COLUMN image_id UUID;
CREATE INDEX idx_collections_image ON collections(image_id) WHERE image_id IS NOT NULL;

-- +goose Down
DROP INDEX idx_collections_image;
ALTER TABLE collections DROP COLUMN image_id;
