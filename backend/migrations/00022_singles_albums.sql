-- +goose Up
-- tracks saved without an album are an artist's singles: they get a Singles album per
-- artist so the library lists them like any other album
INSERT INTO albums (title, artist)
SELECT DISTINCT ON (LOWER(artist)) 'Singles', artist
FROM music_metadata
WHERE album_id IS NULL AND artist <> ''
ORDER BY LOWER(artist), artist
ON CONFLICT (LOWER(title), LOWER(artist)) DO NOTHING;

UPDATE music_metadata mm
SET album_id = a.id
FROM albums a
WHERE mm.album_id IS NULL AND mm.artist <> ''
    AND LOWER(a.title) = 'singles' AND LOWER(a.artist) = LOWER(mm.artist);

-- +goose Down
-- the Singles albums are kept: their tracks can't be told apart from ones filed there by hand
