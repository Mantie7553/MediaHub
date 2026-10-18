import useApi from './useApi'

/**
 * Hook for getting job information from the database, refreshed every 5 seconds
 * @returns a list of jobs, the loading state, the error state, pollError when a refresh
 *  failed while older jobs are still shown, and a function to refetch data
 */
export default function useJobs() {
    const { data, loading, error, pollError, reload } = useApi('/admin/jobs', { initial: [], poll: 5000 })
    return { jobs: data, loading, error, pollError, refetch: (showLoading = true) => reload(!showLoading) }
}
