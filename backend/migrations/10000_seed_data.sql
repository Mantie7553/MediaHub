-- +goose Up
-- Seed data for development
-- Admin password: admin1234

-- ============================================================
-- USERS
-- ============================================================

INSERT INTO users (id, username, email, password_hash, role, download_permission)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'admin',
    'admin@mediahub.local',
    '$2b$14$/NC54dx/yDtD8D2fWAiOLOtV3xU2gDFRv0kbdq2JCJ9PfxNLpu4OO',
    'admin',
    'auto_approved'
);

-- ============================================================
-- MANGA
-- ============================================================

INSERT INTO media_items (id, type, title, description, cover_image_url, release_date, external_id, external_source)
VALUES
    (
        '10000000-0000-0000-0000-000000000001',
        'manga',
        'Berserk',
        'A dark fantasy manga following Guts, a lone mercenary, and his struggle against demonic forces after a traumatic betrayal.',
        'https://cdn.myanimelist.net/images/manga/1/157931.jpg',
        '1989-08-25',
        '2',
        'manual'
    ),
    (
        '10000000-0000-0000-0000-000000000002',
        'manga',
        'Vinland Saga',
        'A historical manga set in the Viking Age following young Thorfinn on a quest for revenge and later, redemption.',
        'https://cdn.myanimelist.net/images/manga/2/188925.jpg',
        '2005-07-13',
        '3',
        'manual'
    ),
    (
        '10000000-0000-0000-0000-000000000003',
        'manga',
        'Vagabond',
        'A fictionalized retelling of the life of Miyamoto Musashi, legendary Japanese swordsman, rendered in stunning artwork.',
        'https://cdn.myanimelist.net/images/manga/1/259070.jpg',
        '1998-09-03',
        '4',
        'manual'
    );

INSERT INTO manga_metadata (media_item_id, total_chapters, genres, status)
VALUES
    ('10000000-0000-0000-0000-000000000001', 364, ARRAY['Dark Fantasy', 'Action', 'Adventure'], 'hiatus'),
    ('10000000-0000-0000-0000-000000000002', 200, ARRAY['Historical', 'Action', 'Drama'],        'ongoing'),
    ('10000000-0000-0000-0000-000000000003', 327, ARRAY['Historical', 'Action', 'Drama'],        'hiatus');

INSERT INTO manga_chapters (id, media_item_id, chapter_number, title, page_count, created_at)
VALUES
    ('c0000000-0000-0000-0001-000000000001', '10000000-0000-0000-0000-000000000001', 1,   'The Black Swordsman',  48, NOW()),
    ('c0000000-0000-0000-0001-000000000002', '10000000-0000-0000-0000-000000000001', 2,   'The Brand',            24, NOW()),
    ('c0000000-0000-0000-0002-000000000001', '10000000-0000-0000-0000-000000000002', 1,   'Normanni',             48, NOW()),
    ('c0000000-0000-0000-0002-000000000002', '10000000-0000-0000-0000-000000000002', 2,   'Thor''s Apples',       24, NOW()),
    ('c0000000-0000-0000-0003-000000000001', '10000000-0000-0000-0000-000000000003', 1,   'Takezo',               56, NOW()),
    ('c0000000-0000-0000-0003-000000000002', '10000000-0000-0000-0000-000000000003', 2,   'Departure',            24, NOW());

-- ============================================================
-- ANIME
-- ============================================================

INSERT INTO media_items (id, type, title, description, cover_image_url, release_date, external_id, external_source)
VALUES
    (
        '20000000-0000-0000-0000-000000000001',
        'anime',
        'Fullmetal Alchemist: Brotherhood',
        'Two brothers use alchemy in a quest to restore their bodies after a failed ritual, uncovering a vast government conspiracy.',
        'https://cdn.myanimelist.net/images/anime/1208/94745.jpg',
        '2009-04-05',
        '5114',
        'anilist'
    ),
    (
        '20000000-0000-0000-0000-000000000002',
        'anime',
        'Steins;Gate',
        'A self-proclaimed mad scientist accidentally discovers time travel and must face the consequences of altering the past.',
        'https://cdn.myanimelist.net/images/anime/5/73199.jpg',
        '2011-04-06',
        '9253',
        'anilist'
    ),
    (
        '20000000-0000-0000-0000-000000000003',
        'anime',
        'Hunter x Hunter (2011)',
        'A young boy sets out to find his missing father and become a Hunter, facing increasingly dangerous challenges.',
        'https://cdn.myanimelist.net/images/anime/1337/99013.jpg',
        '2011-10-02',
        '11061',
        'anilist'
    );

INSERT INTO anime_metadata (media_item_id, studio, status, genres)
VALUES
    ('20000000-0000-0000-0000-000000000001', 'Bones',       'finished', ARRAY['Action', 'Adventure', 'Fantasy']),
    ('20000000-0000-0000-0000-000000000002', 'White Fox',   'finished', ARRAY['Sci-Fi', 'Thriller', 'Drama']),
    ('20000000-0000-0000-0000-000000000003', 'Madhouse',    'finished', ARRAY['Action', 'Adventure', 'Fantasy']);

INSERT INTO anime_seasons (id, media_item_id, season_number, episode_count, title, air_date)
VALUES
    ('a0000000-0000-0001-0000-000000000001', '20000000-0000-0000-0000-000000000001', 1, 64,  'Fullmetal Alchemist: Brotherhood', '2009-04-05'),
    ('a0000000-0000-0002-0000-000000000001', '20000000-0000-0000-0000-000000000002', 1, 24,  'Steins;Gate',                     '2011-04-06'),
    ('a0000000-0000-0003-0000-000000000001', '20000000-0000-0000-0000-000000000003', 1, 148, 'Hunter x Hunter',                 '2011-10-02');

-- ============================================================
-- MOVIES
-- ============================================================

INSERT INTO media_items (id, type, title, description, cover_image_url, release_date, external_id, external_source)
VALUES
    (
        '30000000-0000-0000-0000-000000000001',
        'movie',
        'Interstellar',
        'A team of explorers travel through a wormhole in space in an attempt to ensure humanity''s survival.',
        'https://image.tmdb.org/t/p/w500/gEU2QniE6E77NI6lCU6MxlNBvIx.jpg',
        '2014-11-07',
        '157336',
        'tmdb'
    ),
    (
        '30000000-0000-0000-0000-000000000002',
        'movie',
        'The Grand Budapest Hotel',
        'The adventures of a legendary concierge and his protégé involving a murder, a stolen painting, and a disputed fortune.',
        'https://image.tmdb.org/t/p/w500/eWdyYQreja6JGCzqHWXpWHDrrPo.jpg',
        '2014-03-28',
        '194662',
        'tmdb'
    ),
    (
        '30000000-0000-0000-0000-000000000003',
        'movie',
        'Mad Max: Fury Road',
        'In a post-apocalyptic wasteland, a woman rebels against a tyrant and must outrun his army in a souped-up truck.',
        'https://image.tmdb.org/t/p/w500/hA2ple9q4qnwxp3hKVNhroipsir.jpg',
        '2015-05-15',
        '76341',
        'tmdb'
    );

INSERT INTO movie_metadata (media_item_id, runtime_mins, director, genres)
VALUES
    ('30000000-0000-0000-0000-000000000001', 169, 'Christopher Nolan', ARRAY['Sci-Fi', 'Drama', 'Adventure']),
    ('30000000-0000-0000-0000-000000000002', 100, 'Wes Anderson',      ARRAY['Comedy', 'Drama', 'Mystery']),
    ('30000000-0000-0000-0000-000000000003', 120, 'George Miller',     ARRAY['Action', 'Sci-Fi', 'Adventure']);

-- ============================================================
-- MUSIC
-- ============================================================

INSERT INTO albums (id, title, artist, release_date, external_id, external_source)
VALUES
    (
        'b0000000-0000-0000-0000-000000000001',
        'In Rainbows',
        'Radiohead',
        '2007-10-10',
        'e2f7f0b3-0e2e-4e69-b9c1-81b59b6e8f6d',
        'musicbrainz'
    ),
    (
        'b0000000-0000-0000-0000-000000000002',
        'Currents',
        'Tame Impala',
        '2015-07-17',
        'f2dbac93-13e7-4e91-8d63-e6ad1fc2b6fe',
        'musicbrainz'
    );

INSERT INTO media_items (id, type, title, description, cover_image_url, release_date, external_id, external_source)
VALUES
    (
        '40000000-0000-0000-0000-000000000001',
        'music_track',
        'Weird Fishes / Arpeggi',
        NULL,
        NULL,
        '2007-10-10',
        NULL,
        'manual'
    ),
    (
        '40000000-0000-0000-0000-000000000002',
        'music_track',
        'Reckoner',
        NULL,
        NULL,
        '2007-10-10',
        NULL,
        'manual'
    ),
    (
        '40000000-0000-0000-0000-000000000003',
        'music_track',
        'Let It Happen',
        NULL,
        NULL,
        '2015-07-17',
        NULL,
        'manual'
    ),
    (
        '40000000-0000-0000-0000-000000000004',
        'music_track',
        'The Less I Know the Better',
        NULL,
        NULL,
        '2015-07-17',
        NULL,
        'manual'
    );

INSERT INTO music_metadata (media_item_id, artist, album_id, track_number, duration_secs, genres)
VALUES
    ('40000000-0000-0000-0000-000000000001', 'Radiohead',   'b0000000-0000-0000-0000-000000000001', 4, 318, ARRAY['Alternative', 'Art Rock']),
    ('40000000-0000-0000-0000-000000000002', 'Radiohead',   'b0000000-0000-0000-0000-000000000001', 8, 290, ARRAY['Alternative', 'Art Rock']),
    ('40000000-0000-0000-0000-000000000003', 'Tame Impala', 'b0000000-0000-0000-0000-000000000002', 1, 467, ARRAY['Psychedelic Rock', 'Indie']),
    ('40000000-0000-0000-0000-000000000004', 'Tame Impala', 'b0000000-0000-0000-0000-000000000002', 5, 218, ARRAY['Psychedelic Rock', 'Indie']);

-- +goose Down
DELETE FROM music_metadata      WHERE media_item_id::text LIKE '40000000%';
DELETE FROM media_items         WHERE id::text LIKE '40000000%';
DELETE FROM albums              WHERE id::text LIKE 'b0000000%';
DELETE FROM movie_metadata      WHERE media_item_id::text LIKE '30000000%';
DELETE FROM media_items         WHERE id::text LIKE '30000000%';
DELETE FROM anime_seasons       WHERE media_item_id::text LIKE '20000000%';
DELETE FROM anime_metadata      WHERE media_item_id::text LIKE '20000000%';
DELETE FROM media_items         WHERE id::text LIKE '20000000%';
DELETE FROM manga_chapters      WHERE media_item_id::text LIKE '10000000%';
DELETE FROM manga_metadata      WHERE media_item_id::text LIKE '10000000%';
DELETE FROM media_items         WHERE id::text LIKE '10000000%';
DELETE FROM download_jobs       WHERE request_id IN (SELECT id FROM download_requests WHERE requested_by = '00000000-0000-0000-0000-000000000001');
DELETE FROM download_requests   WHERE requested_by = '00000000-0000-0000-0000-000000000001';
DELETE FROM users               WHERE id = '00000000-0000-0000-0000-000000000001';