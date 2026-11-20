// MangaDex serves these smaller copies of every cover beside the original
const MANGADEX_COVERS = "https://uploads.mangadex.org/covers/"
const MANGADEX_SIZED = /\.(256|512)\.jpg$/
// TMDB's full size posters are several MB; this size is still sharp on the largest cover shown
const TMDB_ORIGINAL = /^https?:\/\/image\.tmdb\.org\/t\/p\/original\//

/**
 * The URL to load a cover from: a smaller copy where the host has one, and MangaDex covers
 * through our proxy, since MangaDex blocks hotlinking
 * @param {string} src the stored cover URL
 * @param {string} source the item's external_source. A manga from MangaDex can have a cover
 *   an admin uploaded, which is served by the backend already
 * @returns {string} the URL for the <img>
 */
export function coverUrl(src, source) {
    if (!src) return src
    if (source === "mangadex" && src.startsWith("https://uploads.mangadex.org/")) {
        const sized = src.startsWith(MANGADEX_COVERS) && !MANGADEX_SIZED.test(src) ? `${src}.512.jpg` : src
        return `/api/proxy/cover?url=${encodeURIComponent(sized)}`
    }
    return src.replace(TMDB_ORIGINAL, "https://image.tmdb.org/t/p/w780/")
}
