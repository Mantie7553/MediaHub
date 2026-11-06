import api from './api'

/**
 * Put a title on the logged in user's list, or change the entry they already have
 * @param {object} [entry] their list entry for it, when they have one
 * @param {object} target what to add when they don't, { media_item_id } or { album_id }
 * @param {object} fields what to save, status and optionally rating
 */
export function saveListEntry(entry, target, fields) {
    return entry
        ? api.put(`/me/media/${entry.id}`, fields)
        : api.post("/me/media", { ...target, ...fields })
}

/**
 * Take a title off the logged in user's list. Only their own entry, the media stays
 * @param {string} entryId the list entry id (user_media_status), not the media id
 */
export function removeFromList(entryId) {
    return api.delete(`/me/media/${entryId}`)
}

/**
 * Admin only: permanently delete a title, its files, its Sonarr/Radarr entry and everyone's
 * lists and progress for it
 * @param {string} id the media item id
 */
export function deleteMedia(id) {
    return api.delete(`/media/${id}`)
}

/**
 * Admin only: permanently delete an album and all of its tracks
 * @param {string} id the album id
 */
export function deleteAlbum(id) {
    return api.delete(`/albums/${id}`)
}

/**
 * Admin only: add a chapter file (CBZ or ZIP) to a manga
 * @param {string} mangaId the manga's media item id
 * @param {File} file the chapter archive
 * @param {string} number the chapter number, read from the file name when empty
 * @param {function} onProgress called with 0 to 1 as the file uploads
 */
export function uploadChapter(mangaId, file, number, onProgress) {
    const form = new FormData()
    form.append("file", file)
    if (number) form.append("number", number)
    return api.post(`/admin/manga/${mangaId}/chapters`, form, {
        onUploadProgress: e => e.total && onProgress?.(e.loaded / e.total),
    })
}

/**
 * Admin only: start a job that downloads the chapters MangaDex and the manga's linked
 * Suwayomi source have that the library doesn't
 * @param {string} mangaId the manga's media item id
 */
export function checkForChapters(mangaId) {
    return api.post(`/admin/manga/${mangaId}/check`)
}

/**
 * Admin only: whether Suwayomi is set up, and the source title a manga is linked to
 * @param {string} mangaId the manga's media item id
 * @returns {Promise} data is { configured, link: { source_name, manga_id, title } | null }
 */
export function getSuwayomiLink(mangaId) {
    return api.get(`/admin/manga/${mangaId}/suwayomi`)
}

/**
 * Admin only: search the installed Suwayomi sources for a title to link a manga to
 * @param {string} mangaId the manga's media item id
 * @param {string} query the search text, the manga's title when empty
 * @returns {Promise} data is { query, sources, failed: [{ source, error }], results: [{ id, title,
 *  source_name, exact, most_recent, checked, chapters: { latest, count } | null }] }. The exact
 *  matches (or the first few results) come checked, newest first
 */
export function searchSuwayomi(mangaId, query) {
    return api.get(`/admin/manga/${mangaId}/suwayomi/search`, { params: query ? { q: query } : {} })
}

/**
 * Admin only: how far one Suwayomi title goes, for search results not looked up with the
 * search and for the linked title
 * @param {string} mangaId the manga's media item id
 * @param {number} suwayomiMangaId the Suwayomi title's id
 * @returns {Promise} data is { chapters: { latest, count } | null }
 */
export function suwayomiChapterStats(mangaId, suwayomiMangaId) {
    return api.get(`/admin/manga/${mangaId}/suwayomi/chapters`, { params: { manga_id: suwayomiMangaId } })
}

/**
 * Admin only: get a manga's missing chapters from this Suwayomi title from now on
 * @param {string} mangaId the manga's media item id
 * @param {number} suwayomiMangaId the search result's id
 */
export function linkSuwayomi(mangaId, suwayomiMangaId) {
    return api.put(`/admin/manga/${mangaId}/suwayomi`, { manga_id: suwayomiMangaId })
}

/**
 * Admin only: stop getting a manga's chapters from Suwayomi
 * @param {string} mangaId the manga's media item id
 */
export function unlinkSuwayomi(mangaId) {
    return api.delete(`/admin/manga/${mangaId}/suwayomi`)
}
