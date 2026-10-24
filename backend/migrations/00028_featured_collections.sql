-- +goose Up
-- featured copies belong to nobody, so no one can change them and deleting an account (even the
-- admin who featured one) doesn't take them down. Everything else still has an owner
ALTER TABLE collections ALTER COLUMN owner_id DROP NOT NULL;
ALTER TABLE collections ADD CONSTRAINT collections_owner_unless_featured
    CHECK (owner_id IS NOT NULL OR visibility = 'featured');

ALTER TABLE collections
    -- the user's playlist or collection this is a copy of; a resubmit replaces the featured copy
    -- of the same original
    ADD COLUMN source_collection_id UUID REFERENCES collections(id) ON DELETE SET NULL,
    -- who made the original, kept for credit
    ADD COLUMN created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN featured_by UUID REFERENCES users(id) ON DELETE SET NULL,
    -- set on the copies that show in the Featured row
    ADD COLUMN featured_at TIMESTAMPTZ,
    -- a playlist copied along with the featured collection it is nested in, and removed with it
    ADD COLUMN part_of UUID REFERENCES collections(id) ON DELETE CASCADE;

CREATE UNIQUE INDEX uq_collections_featured_source ON collections(source_collection_id)
    WHERE featured_at IS NOT NULL;
CREATE INDEX idx_collections_featured ON collections(featured_at) WHERE featured_at IS NOT NULL;
CREATE INDEX idx_collections_part_of ON collections(part_of) WHERE part_of IS NOT NULL;

CREATE TYPE submission_status AS ENUM ('pending', 'approved', 'rejected');

-- a user asking for one of their playlists or collections to be featured
CREATE TABLE collection_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    collection_id UUID NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    submitted_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    note TEXT,
    status submission_status NOT NULL DEFAULT 'pending',
    -- why it was turned down, shown to the user
    admin_note TEXT,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    resolved_by UUID REFERENCES users(id) ON DELETE SET NULL
);
-- one waiting submission per playlist at a time
CREATE UNIQUE INDEX uq_collection_submissions_pending ON collection_submissions(collection_id)
    WHERE status = 'pending';
CREATE INDEX idx_collection_submissions_status ON collection_submissions(status, submitted_at);

-- +goose Down
DROP TABLE collection_submissions;
DROP TYPE submission_status;
DELETE FROM collections WHERE visibility = 'featured';
ALTER TABLE collections
    DROP COLUMN part_of,
    DROP COLUMN featured_at,
    DROP COLUMN featured_by,
    DROP COLUMN created_by,
    DROP COLUMN source_collection_id;
ALTER TABLE collections DROP CONSTRAINT collections_owner_unless_featured;
ALTER TABLE collections ALTER COLUMN owner_id SET NOT NULL;
