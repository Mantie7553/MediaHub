import useApi from './useApi'

/**
 * Load one media item
 * @param {string} id the media item id
 * @returns the item, loading, error, notFound, and refetch(showLoading = true)
 */
export default function useMediaItem(id) {
    const { data, loading, error, notFound, reload } = useApi(`/media/${id}`)
    return { item: data, loading, error, notFound, refetch: (showLoading = true) => reload(!showLoading) }
}
