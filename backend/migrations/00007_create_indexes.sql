-- +goose Up
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_media_items_external ON media_items(external_source, external_id);
CREATE INDEX idx_user_media_status_user ON user_media_status(user_id);
CREATE INDEX idx_user_anime_progress_user ON user_anime_progress(user_id);
CREATE INDEX idx_download_requests_status ON download_requests(status);
CREATE INDEX idx_download_requests_user ON download_requests(requested_by);
CREATE INDEX idx_download_jobs_status ON download_jobs(status);
CREATE INDEX idx_download_jobs_request ON download_jobs(request_id);

-- +goose Down
DROP INDEX idx_download_jobs_request;
DROP INDEX idx_download_jobs_status;
DROP INDEX idx_download_requests_user;
DROP INDEX idx_download_requests_status;
DROP INDEX idx_user_anime_progress_user;
DROP INDEX idx_user_media_status_user;
DROP INDEX idx_media_items_external;
DROP INDEX idx_refresh_tokens_user_id;