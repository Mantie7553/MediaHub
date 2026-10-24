-- +goose NO TRANSACTION

-- +goose Up
-- a copy an admin approved: read-only, and everyone can open it
ALTER TYPE collection_visibility ADD VALUE IF NOT EXISTS 'featured';

-- +goose Down
-- enum values can't be dropped; 'featured' stays unused after rolling back
