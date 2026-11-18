// the ways each kind of list can be sorted. The first option in each set is the list's default.
// An option without compare keeps the order the list came in
import { date, number, text, thenBy, titleOf } from "./sort"

const byName = text(titleOf)

/**
 * The user's list entry for an item: the item itself when it is one (from /me/media or
 * /me/continue), otherwise looked up the way a card does
 */
function entryOf(item, ctx) {
    if (item.media_type !== undefined && item.status !== undefined) return item
    return ctx?.userContentMap?.[String(item.external_id ?? item.media_item_id ?? item.id)]
}

const ratingOf = (item, ctx) => entryOf(item, ctx)?.rating

/**
 * How far through a title, 0 to 1: the part counted against its length when that's known. A
 * finished movie (or anything marked completed) is all the way, and anything else unknown
 */
function progressOf(item, ctx) {
    if (item.progress_pct != null) return item.progress_pct / 100
    const entry = entryOf(item, ctx)
    if (!entry) return null
    const total = entry.progress_total_known ? entry.progress_total : null
    if (total) return Math.min(1, (entry.progress_count ?? 0) / total)
    if (entry.status === "completed") return 1
    return null
}
const partsDone = (item, ctx) => entryOf(item, ctx)?.progress_count

// where each list status sorts under "Status": what's on the go first
const STATUS_ORDER = ["current", "planned", "wishlist", "completed", "dropped"]
const statusRank = item => {
    const index = STATUS_ORDER.indexOf(item.status)
    return index === -1 ? null : index
}

const option = (key, label, compare) => ({ key, label, compare })

const titleAsc = option("title", "Title A–Z", byName)
const titleDesc = option("title-desc", "Title Z–A", text(titleOf, true))
const releaseNew = option("release-new", "Newest release", thenBy(date(i => i.release_date, true), byName))
const releaseOld = option("release-old", "Oldest release", thenBy(date(i => i.release_date), byName))
const ratingHigh = option("rating-high", "Highest rated", thenBy(number(ratingOf, true), byName))
const ratingLow = option("rating-low", "Lowest rated", thenBy(number(ratingOf), byName))
const progressMost = option("progress-most", "Most progress",
    thenBy(number(progressOf, true), number(partsDone, true), byName))
const progressLeast = option("progress-least", "Least progress",
    thenBy(number(progressOf), number(partsDone), byName))
const updated = option("updated", "Recently updated", thenBy(date(i => i.updated_at, true), byName))
const byStatus = option("status", "Status", thenBy(number(statusRank), byName))
const artistAsc = option("artist", "Artist A–Z", thenBy(text(i => i.artist), byName))
const artistDesc = option("artist-desc", "Artist Z–A", thenBy(text(i => i.artist, true), byName))
const nameAsc = option("name", "Name A–Z", byName)
const nameDesc = option("name-desc", "Name Z–A", text(titleOf, true))

/**
 * Titles on the user's list (anime, movies, manga, light novels)
 * @param {string} type the media type; movies have no parts to count, so no progress sorts
 */
export function trackedSorts(type) {
    return [
        titleAsc, titleDesc, releaseNew, releaseOld, ratingHigh, ratingLow,
        ...(type === "movie" ? [] : [progressMost, progressLeast]),
        updated, byStatus,
    ]
}

/** Albums on the user's list */
export const trackedAlbumSorts = [titleAsc, titleDesc, artistAsc, artistDesc, ratingHigh, ratingLow, updated, byStatus]

/** Titles the server has (Available Now), with the user's rating and progress where they track them */
export function librarySorts(type) {
    return [
        titleAsc, titleDesc, releaseNew, releaseOld,
        option("added", "Recently added", thenBy(date(i => i.created_at, true), byName)),
        ratingHigh, ratingLow,
        ...(type === "movie" ? [] : [progressMost, progressLeast]),
    ]
}

/** Albums the server has */
export const albumSorts = [
    titleAsc, titleDesc,
    option("artist", "Artist A–Z", thenBy(text(i => i.artist), releaseOld.compare)),
    option("artist-desc", "Artist Z–A", thenBy(text(i => i.artist, true), releaseOld.compare)),
    releaseNew, releaseOld,
    option("added", "Recently added", thenBy(date(i => i.created_at, true), byName)),
    option("tracks", "Most tracks", thenBy(number(i => i.track_count, true), byName)),
    ratingHigh, ratingLow,
]

/** Titles the user is partway through */
export const continueSorts = [
    option("recent", "Recently watched or read"),
    titleAsc, titleDesc, progressMost, progressLeast, releaseNew, releaseOld,
]

/** The user's playlists or collections */
export const collectionSorts = [
    option("updated", "Recently changed"),
    nameAsc, nameDesc,
    option("size", "Most items", thenBy(number(i => i.item_count, true), byName)),
    option("size-asc", "Fewest items", thenBy(number(i => i.item_count), byName)),
]

/** Playlists admins featured */
export const featuredSorts = [option("featured", "Newest featured"), nameAsc, nameDesc,
    option("size", "Most items", thenBy(number(i => i.item_count, true), byName))]

/** One playlist per artist */
export const artistSorts = [nameAsc, nameDesc,
    option("tracks", "Most tracks", thenBy(number(i => i.item_count, true), byName)),
    option("tracks-asc", "Fewest tracks", thenBy(number(i => i.item_count), byName))]

/**
 * Results from a search or a source's own lists (trending, popular, top rated)
 * @param {string} first what the source's own order is called, like "Best match" or "Trending"
 */
export function resultSorts(first) {
    return [option("source", first), titleAsc, titleDesc, ratingHigh, ratingLow]
}

/** Songs and videos from a music search */
export const musicResultSorts = [
    option("source", "Best match"), titleAsc, titleDesc,
    option("artist", "Artist A–Z", thenBy(text(i => i.artist || i.uploader), byName)),
    option("short", "Shortest", thenBy(number(i => i.duration_secs), byName)),
    option("long", "Longest", thenBy(number(i => i.duration_secs, true), byName)),
]

// a playlist's tracks, ordered by album and then where they sit in it
const albumThenTrack = thenBy(text(i => i.album_title), number(i => i.track_number), byName)

/** The tracks of a playlist. "Custom order" is the owner's own */
export const playlistSorts = [
    option("custom", "Custom order"),
    titleAsc, titleDesc,
    option("artist", "Artist", thenBy(text(i => i.artist), byName)),
    option("album", "Album", albumThenTrack),
    option("artist-album", "Artist, then album", thenBy(text(i => i.artist), date(i => i.release_date), albumThenTrack)),
    option("album-artist", "Album, then artist", thenBy(text(i => i.album_title), text(i => i.artist), number(i => i.track_number), byName)),
    option("added-new", "Recently added", date(i => i.added_at, true)),
    option("added-old", "Oldest added", date(i => i.added_at)),
    releaseNew, releaseOld,
    option("short", "Shortest", thenBy(number(i => i.duration_secs), byName)),
    option("long", "Longest", thenBy(number(i => i.duration_secs, true), byName)),
]

// what each kind of entry in a collection sorts under "Type"
const TYPE_ORDER = ["anime", "movie", "manga", "light_novel", "album", "music_track", "playlist"]

/** What's in a collection. "Custom order" is the owner's own */
export const collectionEntrySorts = [
    option("custom", "Custom order"),
    titleAsc, titleDesc,
    option("type", "Type", thenBy(number(i => TYPE_ORDER.indexOf(i.type)), byName)),
    option("added-new", "Recently added", date(i => i.added_at, true)),
    option("added-old", "Oldest added", date(i => i.added_at)),
    releaseNew, releaseOld,
    ratingHigh, ratingLow,
]

/** An album's tracks */
export const albumTrackSorts = [
    option("track", "Track number"),
    titleAsc, titleDesc,
    option("short", "Shortest", thenBy(number(i => i.duration_secs), byName)),
    option("long", "Longest", thenBy(number(i => i.duration_secs, true), byName)),
]

/**
 * Episodes, chapters or volumes. The comparators get { numberOf, isDone } in their ctx, since
 * each page numbers them and tracks what's done its own way
 * @param {string} undone what a part not done yet is called, like "Unwatched" or "Unread"
 */
export function partSorts(undone) {
    const byNumber = (desc = false) => number((i, ctx) => ctx.numberOf(i), desc)
    return [
        option("oldest", "Oldest first", byNumber()),
        option("newest", "Newest first", byNumber(true)),
        option("undone", `${undone} first`, thenBy(number((i, ctx) => ctx.isDone(i) ? 1 : 0), byNumber())),
    ]
}
