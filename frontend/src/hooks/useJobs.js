import useApi from './useApi'

/**
 * Hook for getting job information from the database, refreshed every 5 seconds
 * @param {boolean} archived the archived jobs instead, which don't change so aren't refreshed
 * @returns a list of jobs, the loading state, the error state, pollError when a refresh
 *  failed while older jobs are still shown, and a function to refetch data
 */
export default function useJobs(archived = false) {
    const { data, loading, error, pollError, reload } = useApi(archived ? '/admin/jobs?archived=true' : '/admin/jobs',
        { initial: [], poll: archived ? 0 : 5000 })
    return { jobs: data, loading, error, pollError, refetch: (showLoading = true) => reload(!showLoading) }
}
