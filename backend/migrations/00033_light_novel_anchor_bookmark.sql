-- +goose Up
-- where the reader was, as the paragraph at the top of the screen and the character offset in
-- it, so reopening a volume lands on the same line however late its images load. The bookmark
-- is the paragraph a reader's orange line sits above, and reopening goes there while it's set
ALTER TABLE light_novel_progress
    ADD COLUMN anchor_block INT,
    ADD COLUMN anchor_char INT,
    ADD COLUMN bookmark_block INT;

-- +goose Down
ALTER TABLE light_novel_progress
    DROP COLUMN bookmark_block,
    DROP COLUMN anchor_char,
    DROP COLUMN anchor_block;
