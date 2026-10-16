-- +goose Up
ALTER TABLE users
    ADD COLUMN preferred_audio_language TEXT NOT NULL DEFAULT 'jpn',
    ADD COLUMN preferred_subtitle_language TEXT NOT NULL DEFAULT 'eng';

-- +goose Down
ALTER TABLE users
    DROP COLUMN preferred_audio_language,
    DROP COLUMN preferred_subtitle_language;
