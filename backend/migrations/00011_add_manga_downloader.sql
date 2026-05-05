-- +goose Up
-- +goose NO TRANSACTION
ALTER TYPE download_handler ADD VALUE IF NOT EXISTS 'mangal';

-- +goose Down