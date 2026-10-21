-- +goose Up
-- fields an admin set by hand (title, cover, genres, ...); library syncs and saves from
-- search leave them alone until the admin hands them back
ALTER TABLE media_items ADD COLUMN locked_fields TEXT[] NOT NULL DEFAULT '{}';
-- Sonarr renames episodes on every sync, so an edited title is kept apart
ALTER TABLE episodes ADD COLUMN title_locked BOOL NOT NULL DEFAULT FALSE;
ALTER TABLE albums ADD COLUMN description TEXT;

-- +goose Down
ALTER TABLE albums DROP COLUMN description;
ALTER TABLE episodes DROP COLUMN title_locked;
ALTER TABLE media_items DROP COLUMN locked_fields;
