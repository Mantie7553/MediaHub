-- +goose Up
CREATE TABLE manga_metadata (
    media_item_id UUID PRIMARY KEY NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    total_chapters INT,
    genres TEXT[],
    status manga_status
);

CREATE TABLE manga_chapters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    media_item_id UUID NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    chapter_number NUMERIC,
    title TEXT,
    file_path TEXT,
    page_count INT,
    created_at TIMESTAMPTZ,
    UNIQUE(media_item_id, chapter_number)
);

CREATE TABLE manga_progress (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    chapter_id UUID REFERENCES manga_chapters(id) ON DELETE CASCADE,
    media_item_id UUID NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    last_page_read INT,
    completed BOOL,
    updated_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, chapter_id)
);

-- +goose Down
DROP TABLE manga_progress;
DROP TABLE manga_chapters;
DROP TABLE manga_metadata;