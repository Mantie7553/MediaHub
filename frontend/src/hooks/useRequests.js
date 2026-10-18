import useApi from './useApi'

/**
 * Hook for getting the user's download requests from the database
 * @returns a list of requests, the loading state, the error state, and a function to refetch the data
 */
export default function useRequests() {
    const { data, loading, error, reload } = useApi('/requests', { initial: [] })
    return { requests: data, loading, error, refetch: (showLoading = true) => reload(!showLoading) }
}
