// a title not in the library yet is only known by its source's id, which can clash across sources
const externalKey = (source, id) => (source && id != null ? `${source}:${id}` : null)

/**
 * The keys a wishlist can be looked up by: each title's media id and its source's id
 * @param {object[]} entries the wishlist from /me/wishlist
 * @returns {Map<string, string>} key → the title's media id
 */
export function wishlistKeys(entries) {
    const keys = new Map()
    for (const entry of entries) {
        keys.set(entry.media_item_id, entry.media_item_id)
        const external = externalKey(entry.external_source, entry.external_id)
        if (external) keys.set(external, entry.media_item_id)
    }
    return keys
}

/**
 * The media id of a title on the wishlist, or undefined when it isn't on it
 * @param {Map<string, string>} keys from wishlistKeys
 * @param {object} item a card's item: a library title, list entry or search result
 */
export function wishlistedId(keys, item) {
    if (!item) return undefined
    const mediaId = item.media_item_id ?? item.id
    if (mediaId && keys.has(mediaId)) return keys.get(mediaId)
    const external = externalKey(item.external_source, item.external_id)
    return external ? keys.get(external) : undefined
}

// the title types wishlisting downloads (matches downloader.WishlistDownloads)
export const WISHLIST_TYPES = ["anime", "movie", "manga"]
