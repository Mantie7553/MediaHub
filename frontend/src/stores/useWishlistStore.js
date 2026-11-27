import { create } from "zustand"
import api from "../services/api"
import { wishlistKeys } from "../utils/wishlist"

/**
 * The logged in user's wishlist: the titles they want downloaded. It is kept apart from their
 * list status, so wishlisting a title never changes where they are with it
 */
const useWishlistStore = create(set => ({
    // the wishlist from /me/wishlist, newest first, shaped like library titles for cards
    entries: [],
    keys: new Map(),
    // until the first answer, so rows can show their loading state
    loading: true,
    error: null,
    load: () => api.get("/me/wishlist")
        .then(res => {
            const entries = res.data ?? []
            set({ entries, keys: wishlistKeys(entries), loading: false, error: null })
        })
        // a failed refresh keeps the wishlist already shown
        .catch(error => set(state => ({ loading: false, error: state.loading ? error : state.error }))),
}))

export function loadWishlist() {
    return useWishlistStore.getState().load()
}

/**
 * Put a title on the wishlist, which asks for it to be downloaded
 * @param {object} item the title; one not in the library yet is saved from its search result
 * @returns {Promise<{download?: string, reason?: string}>} what the server did about the download
 */
export function addToWishlist(item) {
    const mediaId = item.media_item_id ?? item.id
    const request = mediaId
        ? api.put(`/me/wishlist/${mediaId}`)
        : api.post("/search/save", {
            external_id: item.external_id,
            external_source: item.external_source,
            title: item.title ?? item.media_title,
            cover_image_url: item.cover_image_url,
            type: item.type ?? item.media_type,
            action: "wishlist",
            tvdb_id: item.tvdb_id,
        })
    return request.then(res => loadWishlist().then(() => res.data ?? {}))
}

/**
 * Take a title off the wishlist. A download already asked for carries on
 * @param {string} mediaId the title's media id
 */
export function removeFromWishlist(mediaId) {
    return api.delete(`/me/wishlist/${mediaId}`).then(loadWishlist)
}

export default useWishlistStore
