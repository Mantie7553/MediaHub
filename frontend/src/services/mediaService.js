import api from './api'

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
