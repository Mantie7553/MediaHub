import api from './api'

/*
Admin only: fixing the library by hand. Fields saved here are kept by the server through
library syncs until they are handed back (see backend/internal/locks)
*/

/**
 * Change a title's details. Only the fields given change; null clears one
 * @param {string} id the media item id
 * @param {object} changes { title, description, release_date, genres, studio, ... }
 * @param {string[]} unlock fields to hand back to the library syncs
 */
export function editMedia(id, changes, unlock = []) {
    return api.put(`/admin/media/${id}`, { ...changes, unlock })
}

/**
 * Replace a title's cover
 * @param {string} id the media item id
 * @param {File} file a JPEG, PNG, WebP or GIF
 * @returns the response, with data.cover_image_url signed for <img>
 */
export function uploadMediaCover(id, file) {
    const form = new FormData()
    form.append("file", file)
    return api.post(`/admin/media/${id}/cover`, form)
}

/**
 * Replace an album's cover
 * @param {string} id the album id
 * @param {File} file a JPEG, PNG, WebP or GIF
 */
export function uploadAlbumCover(id, file) {
    const form = new FormData()
    form.append("file", file)
    return api.post(`/admin/albums/${id}/cover`, form)
}

/**
 * Change an album's title, artist, release date or description
 * @param {string} id the album id
 * @param {object} changes the fields to change
 */
export function editAlbum(id, changes) {
    return api.put(`/admin/albums/${id}`, changes)
}

// the endpoints for each piece of a title; tracks are titles of their own
const pieceTypes = { episode: "episodes", chapter: "chapters", volume: "volumes" }

/**
 * Rename or renumber an episode, chapter or volume
 * @param {string} kind episode, chapter or volume
 * @param {string} id its id
 * @param {object} changes { title } and { chapter_number } or { volume_number }
 */
export function editPiece(kind, id, changes) {
    return api.put(`/admin/${pieceTypes[kind]}/${id}`, changes)
}

/**
 * Delete an episode, chapter or volume and its file. Tracks go through deleteMedia
 * @param {string} kind episode, chapter or volume
 * @param {string} id its id
 */
export function deletePiece(kind, id) {
    return api.delete(`/admin/${pieceTypes[kind]}/${id}`)
}

// pieces sent per request: under Cloudflare's 100 MB limit, and small enough that a retry
// doesn't resend much
const CHUNK_BYTES = 32 * 1024 * 1024
const CHUNK_TRIES = 4

/**
 * Send a file to the server in chunks, retrying a chunk that fails and carrying on from what
 * the server already has
 * @param {File} file the file
 * @param {string} kind video, audio or epub
 * @param {function} onProgress called with 0 to 1
 * @param {AbortSignal} signal stops the upload
 * @returns the upload id, to name in the request that adds it
 */
export async function uploadInChunks(file, kind, onProgress, signal) {
    const { data } = await api.post("/admin/uploads", { name: file.name, size: file.size, kind }, { signal })
    const id = data.id
    const chunkSize = Math.min(CHUNK_BYTES, data.max_chunk || CHUNK_BYTES)
    let offset = 0
    let failures = 0

    try {
        while (offset < file.size) {
            const end = Math.min(offset + chunkSize, file.size)
            try {
                const res = await api.put(`/admin/uploads/${id}`, file.slice(offset, end), {
                    signal,
                    headers: { "Content-Type": "application/octet-stream", "Upload-Offset": String(offset) },
                    onUploadProgress: e => onProgress?.((offset + (e.loaded ?? 0)) / file.size),
                })
                offset = res.data.received
                failures = 0
            } catch (err) {
                // the server answers 409 with what it has when a retry overlaps
                const received = err.response?.status === 409 ? err.response.data?.received : undefined
                if (typeof received === "number") {
                    offset = received
                    continue
                }
                if (signal?.aborted || err.response?.status < 500 || ++failures >= CHUNK_TRIES) throw err
                await new Promise(r => setTimeout(r, 1000 * failures))
            }
            onProgress?.(offset / file.size)
        }
    } catch (err) {
        api.delete(`/admin/uploads/${id}`).catch(() => {})
        throw err
    }
    return id
}

/**
 * Forget an upload that won't be used
 * @param {string} uploadId from uploadInChunks
 */
export function cancelUpload(uploadId) {
    return api.delete(`/admin/uploads/${uploadId}`)
}

/**
 * Add an uploaded episode file to an anime
 * @param {string} animeId the anime's media item id
 * @param {string} uploadId from uploadInChunks
 * @param {object} details { season_number, episode_number, title }
 */
export function addEpisode(animeId, uploadId, details) {
    return api.post(`/admin/anime/${animeId}/episodes`, { upload_id: uploadId, ...details })
}

/**
 * Give a movie without a file the uploaded one
 * @param {string} movieId the movie's media item id
 * @param {string} uploadId from uploadInChunks
 */
export function addMovieFile(movieId, uploadId) {
    return api.post(`/admin/movies/${movieId}/file`, { upload_id: uploadId })
}

/**
 * Add an uploaded EPUB to a light novel
 * @param {string} novelId the light novel's media item id
 * @param {string} uploadId from uploadInChunks
 * @param {object} details { volume_number, title }
 */
export function addVolume(novelId, uploadId, details) {
    return api.post(`/admin/light-novels/${novelId}/volumes`, { upload_id: uploadId, ...details })
}

/**
 * Add an uploaded mp3 to an album
 * @param {string} albumId the album id
 * @param {string} uploadId from uploadInChunks
 * @param {object} details { title, track_number }
 */
export function addTrack(albumId, uploadId, details) {
    return api.post(`/admin/albums/${albumId}/tracks`, { upload_id: uploadId, ...details })
}
