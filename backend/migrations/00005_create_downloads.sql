-- +goose Up
CREATE TABLE download_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    requested_by UUID REFERENCES users(id) ON DELETE SET NULL,
    media_item_id UUID REFERENCES media_items(id) ON DELETE SET NULL,
    album_id UUID REFERENCES albums(id) ON DELETE SET NULL,
    title_override TEXT,
    source_url TEXT,
    status request_status NOT NULL DEFAULT 'pending',
    auto_approved BOOLEAN NOT NULL DEFAULT FALSE,
    admin_notes TEXT,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    CHECK (
        (media_item_id IS NOT NULL AND album_id IS NULL) OR
        (album_id IS NOT NULL AND media_item_id IS NULL) OR
        (media_item_id IS NULL AND album_id IS NULL AND title_override IS NOT NULL)
    ),
    UNIQUE (requested_by, media_item_id)
);

CREATE TABLE download_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id UUID NOT NULL REFERENCES download_requests(id),
    media_item_id UUID NOT NULL REFERENCES media_items(id),
    destination_path TEXT NOT NULL,
    source_url TEXT NOT NULL,
    status job_status NOT NULL DEFAULT 'queued',
    progress_pct INT NOT NULL DEFAULT 0 CHECK (progress_pct BETWEEN 0 AND 100),
    error_message TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE download_jobs;
DROP TABLE download_requests;