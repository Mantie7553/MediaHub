-- +goose Up
CREATE TYPE user_role AS ENUM ('admin', 'user');
CREATE TYPE download_permission AS ENUM ('vetted', 'auto_approved');
CREATE TYPE media_type AS ENUM ('anime', 'movie', 'manga', 'music_track');
CREATE TYPE external_source AS ENUM ('anilist', 'tmdb', 'musicbrainz', 'manual', 'mangadex');
CREATE TYPE anime_status AS ENUM ('airing', 'finished', 'upcoming');
CREATE TYPE media_status AS ENUM ('watching', 'completed', 'wishlist', 'dropped', 'plan_to_watch', 'listening', 'manga_reading');
CREATE TYPE request_status AS ENUM ('pending', 'approved', 'rejected', 'queued', 'downloading', 'complete', 'failed');
CREATE TYPE job_status AS ENUM ('queued', 'downloading', 'complete', 'failed');
CREATE TYPE manga_status AS ENUM ('ongoing', 'completed', 'hiatus');

-- +goose Down
DROP TYPE manga_status;
DROP TYPE job_status;
DROP TYPE request_status;
DROP TYPE media_status;
DROP TYPE anime_status;
DROP TYPE external_source;
DROP TYPE media_type;
DROP TYPE download_permission;
DROP TYPE user_role;