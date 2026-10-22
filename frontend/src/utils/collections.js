// the two playlists that fill themselves from the user's list, and what fills them
export const SMART_PLAYLISTS = {
    listened: "Tracks and albums you've marked listened.",
    unlistened: "Tracks and albums you plan to listen to.",
}

/**
 * What the rules (backend/internal/handlers/collections/collections_rules.go) call an item
 * @param {object} item a title, track, list entry, album or playlist
 * @returns a media type ("music_track", "anime", ...), "album", "playlist" or "collection"
 */
export function itemKind(item) {
    if (item.kind) return item.kind
    const type = item.type ?? item.media_type
    return type === "music_album" ? "album" : type
}

/**
 * Whether a playlist or collection can take an item, as the server decides. Albums go into a
 * playlist as their tracks
 * @param {string} kind "playlist" or "collection"
 * @param {string} item an itemKind
 */
export function canHold(kind, item) {
    if (kind === "playlist") return item === "music_track" || item === "album"
    if (kind === "collection") return ["anime", "movie", "manga", "light_novel", "music_track", "album", "playlist"].includes(item)
    return false
}

/**
 * The body for adding an item, naming it the way the server expects
 * @param {object} item as for itemKind, with its id (media_item_id or id)
 */
export function addBody(item) {
    const kind = itemKind(item)
    if (kind === "album") return { album_id: item.album_id ?? item.id }
    if (kind === "playlist" || kind === "collection") return { collection_id: item.id }
    return { media_item_id: item.media_item_id ?? item.id }
}

/**
 * Where an entry goes when it's moved by a step: the entry it should end up below, or null for
 * the top. Undefined when it can't move that way
 * @param {object[]} entries the entries in order, with entry_id
 * @param {number} index the entry being moved
 * @param {"up"|"down"|"top"} direction
 */
export function moveTarget(entries, index, direction) {
    if (direction === "top" || (direction === "up" && index === 1)) return index > 0 ? null : undefined
    if (direction === "up") return index > 1 ? entries[index - 2].entry_id : undefined
    if (direction === "down") return index < entries.length - 1 ? entries[index + 1].entry_id : undefined
    return undefined
}

/**
 * The target for a drop: the entry an entry dragged from one place to another ends up below,
 * or null for the top
 * @param {object[]} entries the entries in order
 * @param {number} from where it was
 * @param {number} to where it was dropped, as an index into the list as it was
 */
export function dropTarget(entries, from, to) {
    if (from === to) return undefined
    if (to === 0) return null
    // moving down it lands below the one it was dropped on; moving up, below the one before
    return to > from ? entries[to].entry_id : entries[to - 1].entry_id
}

/**
 * The entries in their new order after a move, so the list can change before the server answers
 * @param {object[]} entries the entries in order
 * @param {number} index the entry that moved
 * @param {string|null} after the entry it went below, null for the top
 */
export function applyMove(entries, index, after) {
    const moving = entries[index]
    const rest = entries.filter((_, i) => i !== index)
    const at = after === null ? 0 : rest.findIndex(e => e.entry_id === after) + 1
    return [...rest.slice(0, at), moving, ...rest.slice(at)]
}

/**
 * A shuffled copy, for Shuffle
 * @param {any[]} list
 * @param {function} random a number from 0 to 1, Math.random unless a test needs a fixed order
 */
export function shuffled(list, random = Math.random) {
    const copy = [...list]
    for (let i = copy.length - 1; i > 0; i--) {
        const j = Math.floor(random() * (i + 1));
        [copy[i], copy[j]] = [copy[j], copy[i]]
    }
    return copy
}
