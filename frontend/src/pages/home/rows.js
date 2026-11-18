/**
 * The user's tracked titles of one type, in no set order (each row sorts itself). Albums come
 * back shaped like /albums so AlbumList can show them, keeping what they sort by
 * @param {object[]} userContent entries from /me/media
 * @param {string} type a media type, music_album for albums
 */
export function trackedOfType(userContent, type) {
    const entries = userContent.filter(item => item.media_type === type);
    const shaped = type === "music_album"
        ? entries.map(item => ({
            id: item.album_id,
            title: item.media_title,
            artist: item.artist,
            cover_image_url: item.cover_image_url,
            track_count: null,
            status: item.status,
            rating: item.rating,
            updated_at: item.updated_at,
            media_type: item.media_type,
        }))
        : entries;
    return shaped;
}

/**
 * The user's playlists and collections from /me/collections. The playlists that fill themselves
 * are always there, so they count only once something is in them
 * @param {object[]} collections summaries from /me/collections
 */
export function splitCollections(collections) {
    return {
        playlists: collections.filter(c => c.kind === "playlist" && (!c.smart || c.item_count > 0)),
        collections: collections.filter(c => c.kind === "collection"),
    };
}
