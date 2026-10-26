import api from './api'

// what to tell when something is added to a playlist or collection, or a new one is made, from
// anywhere: the pickers on cards and pages change what other parts of the screen show
const listeners = new Set()

/**
 * Hear about playlists and collections changing, to reload what shows them
 * @param {function} listener called after something is added or a new one is made
 * @returns a function that stops listening, for an effect's cleanup
 */
export function onCollectionsChanged(listener) {
    listeners.add(listener)
    return () => { listeners.delete(listener) }
}

function changed(res) {
    listeners.forEach(listener => listener())
    return res
}

/**
 * The user's playlists and collections, playlists starting with the two that fill themselves
 * @param {string} kind "playlist", "collection", or nothing for both
 * @param {string} contains an item id; the ones already holding it come back with contains set
 * @param {number} season with contains, an anime's season: only the ones holding that season
 */
export function listCollections(kind, contains, season) {
    return api.get("/me/collections", { params: { kind: kind || undefined, contains: contains || undefined, season: season ?? undefined } })
}

/**
 * Make a new, empty playlist or collection
 * @returns the response, with { id }
 */
export function createCollection(kind, name, description) {
    return api.post("/me/collections", { kind, name, description }).then(changed)
}

/**
 * Rename it, or change its description
 * @param {object} changes { name?, description? }
 */
export function updateCollection(id, changes) {
    return api.put(`/collections/${id}`, changes)
}

export function deleteCollection(id) {
    return api.delete(`/collections/${id}`)
}

/**
 * Add something on the end. An album added to a playlist adds its downloaded tracks
 * @param {object} item exactly one of { media_item_id }, { album_id }, { collection_id }
 * @returns the response, with { added, already }
 */
export function addToCollection(id, item) {
    return api.post(`/collections/${id}/items`, item).then(changed)
}

/**
 * Take one entry out
 * @param {string} entryId the entry's own id (entry_id), not the item's
 */
export function removeFromCollection(id, entryId) {
    return api.delete(`/collections/${id}/items/${entryId}`)
}

/**
 * Move an entry below another, or to the top
 * @param {string|null} after the entry it goes below, null for the top
 */
export function moveInCollection(id, entryId, after) {
    return api.put(`/collections/${id}/items/${entryId}/position`, { after })
}

/**
 * Ask the admins to feature one of the user's playlists or collections
 * @param {string} note an optional message for the admins
 * @returns the response, with where the request is ({ status, ... })
 */
export function submitForFeaturing(id, note) {
    return api.post(`/collections/${id}/submission`, { note: note || undefined })
}

// take back a request that is still waiting
export function withdrawSubmission(id) {
    return api.delete(`/collections/${id}/submission`)
}

/**
 * Save a featured playlist or collection as the user's own copy, which they can change
 * @returns the response, with the copy's { id, kind }
 */
export function saveCopy(id) {
    return api.post(`/collections/${id}/copy`)
}

/**
 * Admins: approve a submission, featuring it as it is now
 * @returns the response, with { featured_id }
 */
export function approveSubmission(submissionId) {
    return api.post(`/admin/collection-submissions/${submissionId}/approve`)
}

/**
 * Admins: turn a submission down
 * @param {string} note why, shown to the user
 */
export function rejectSubmission(submissionId, note) {
    return api.post(`/admin/collection-submissions/${submissionId}/reject`, { note: note || undefined })
}

/**
 * Admins: feature one of their own straight away, or update its featured copy
 * @returns the response, with { featured_id }
 */
export function featureNow(id) {
    return api.post(`/admin/collections/${id}/feature`)
}
