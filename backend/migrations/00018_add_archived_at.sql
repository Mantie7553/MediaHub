-- +goose Up
-- archived requests and jobs are hidden from the admin's Downloads lists but kept as a record
ALTER TABLE download_requests ADD COLUMN archived_at TIMESTAMPTZ;
ALTER TABLE download_jobs ADD COLUMN archived_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE download_jobs DROP COLUMN archived_at;
ALTER TABLE download_requests DROP COLUMN archived_at;
