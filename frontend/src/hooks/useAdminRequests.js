import useApi from './useApi'

/**
 * Hook for getting every user's download requests from the database
 * @param {boolean} archived the archived requests instead
 * @returns a list of requests, the loading state, the error state, and a function to refetch the data
 */
export default function useAdminRequests(archived = false) {
    const { data, loading, error, reload } = useApi(archived ? '/requests/all?archived=true' : '/requests/all', { initial: [] })
    return { requests: data, loading, error, refetch: (showLoading = true) => reload(!showLoading) }
}
