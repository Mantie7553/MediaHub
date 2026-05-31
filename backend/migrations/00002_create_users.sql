-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role user_role NOT NULL DEFAULT 'user',
    download_permission download_permission NOT NULL DEFAULT 'vetted',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO users (username, email, password_hash, role, download_permission)
VALUES ('admin', 'admin@example.com', '$2a$10$hP.c96FjCVog/ktlMbje.OLhbNDt1hmwbor41Ou6fLLSGchF5SRpi', 'admin', 'auto_approved');

-- +goose Down
DROP TABLE refresh_tokens;
DROP TABLE users;