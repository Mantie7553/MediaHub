-- +goose Up
-- the titles a user wants downloaded, kept apart from their list status so wishlisting a title
-- never changes how far they say they are through it
CREATE TABLE IF NOT EXISTS user_wishlist (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    media_item_id UUID NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, media_item_id)
);

-- entries saved with the old "wishlist" status stay wishlisted, with no list status
INSERT INTO user_wishlist (user_id, media_item_id, created_at)
SELECT user_id, media_item_id, updated_at FROM user_media_status
WHERE status = 'wishlist' AND media_item_id IS NOT NULL
ON CONFLICT DO NOTHING;
DELETE FROM user_media_status WHERE status = 'wishlist';

-- +goose Down
INSERT INTO user_media_status (user_id, media_item_id, status, updated_at)
SELECT user_id, media_item_id, 'wishlist', created_at FROM user_wishlist
ON CONFLICT DO NOTHING;
DROP TABLE IF EXISTS user_wishlist;
