-- +goose Up
ALTER TABLE movie_metadata ADD COLUMN file_path TEXT;

-- +goose Down
ALTER TABLE movie_metadata DROP COLUMN file_path;