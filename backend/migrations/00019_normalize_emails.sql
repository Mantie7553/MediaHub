-- +goose Up
-- emails are now stored trimmed and lower case and looked up the same way, so the case someone
-- typed at sign up doesn't matter. An address that would collide with another account's is
-- left as it is (login still finds it, comparing in lower case)
UPDATE users u SET email = LOWER(TRIM(u.email))
WHERE u.email <> LOWER(TRIM(u.email))
AND NOT EXISTS (
    SELECT 1 FROM users o WHERE o.id <> u.id AND LOWER(TRIM(o.email)) = LOWER(TRIM(u.email))
);

-- +goose Down
-- the original case isn't kept, so there is nothing to undo
SELECT 1;
