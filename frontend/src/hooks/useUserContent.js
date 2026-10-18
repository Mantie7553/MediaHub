import useApi from "./useApi";

/**
 * The media the logged in user tracks, with a map to look entries up by media, external or album id
 * @returns userContent, userContentMap, loading, error, refresh (reloads in the background)
 *  and retry (reloads showing the loading state, after an error)
 */
export default function useUserContent() {
    const { data: userContent, loading, error, reload } = useApi("/me/media", { initial: [] });

    const userContentMap = Object.fromEntries([
        ...userContent.map(item => [item.media_item_id, item]),
        ...userContent.filter(item => item.external_id != null).map(item => [String(item.external_id), item]),
        ...userContent.filter(item => item.album_id != null).map(item => [item.album_id, item]),
    ]);

    return { userContent, userContentMap, loading, error, refresh: () => reload(true), retry: () => reload() };
}
