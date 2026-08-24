-- +goose Up
CREATE UNIQUE INDEX album_artist_title ON albums (LOWER(title), LOWER(artist));

-- +goose Down
DROP INDEX album_artist_title;