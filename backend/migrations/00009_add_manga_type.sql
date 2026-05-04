-- +goose Up
-- +goose NO TRANSACTION
ALTER TYPE media_type ADD VALUE IF NOT EXISTS 'manga';
ALTER TYPE media_status ADD VALUE IF NOT EXISTS 'manga_reading';
CREATE TYPE manga_status AS ENUM ('ongoing', 'completed', 'hiatus');

-- +goose Down
-- can't remove enum values in postgres, so just drop the new type
DROP TYPE manga_status;