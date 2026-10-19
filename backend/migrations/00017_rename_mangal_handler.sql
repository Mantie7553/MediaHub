-- +goose Up
-- manga is now downloaded straight from MangaDex instead of through the mangal CLI
ALTER TYPE download_handler RENAME VALUE 'mangal' TO 'mangadex';

-- +goose Down
ALTER TYPE download_handler RENAME VALUE 'mangadex' TO 'mangal';
