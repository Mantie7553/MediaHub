import { useEffect, useMemo } from "react";
import useApi from "./useApi";
import { loadWishlist } from "../stores/useWishlistStore";

/**
 * The media the logged in user tracks, with a map to look entries up by media, external or album id.
 * Also loads their wishlist (stores/useWishlistStore), which cards and buttons read from
 * @returns userContent, userContentMap, loading, error, refresh (reloads in the background)
 *  and retry (reloads showing the loading state, after an error)
 */
export default function useUserContent() {
    const { data: userContent, loading, error, reload } = useApi("/me/media", { initial: [] });
    useEffect(() => { loadWishlist(); }, []);

    // built once per load, not on every render of a page showing several lists
    const userContentMap = useMemo(() => Object.fromEntries([
        ...userContent.map(item => [item.media_item_id, item]),
        ...userContent.filter(item => item.external_id != null).map(item => [String(item.external_id), item]),
        ...userContent.filter(item => item.album_id != null).map(item => [item.album_id, item]),
    ]), [userContent]);

    return { userContent, userContentMap, loading, error, refresh: () => { reload(true); loadWishlist(); }, retry: () => { reload(); loadWishlist(); } };
}
