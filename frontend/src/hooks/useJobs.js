import { useState, useEffect } from 'react'
import api from '../services/api'

/**
 * Hook for getting job information from the database
 * that sets loading and error states
 * @returns a list of jobs, the loading state, the error state, and a function to refetch data
 */
export default function useJobs() {
    const [jobs, setJobs] = useState([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState(null)

    function fetch(showLoading = true) {
        if (showLoading) setLoading(true)
        api.get('/admin/jobs')
            .then(res => setJobs(res.data))
            .catch(err => setError(err))
            .finally(() => setLoading(false))
    }

    useEffect(() => {
        fetch(true)
        const interval = setInterval(() => fetch(false), 5000)
        return () => clearInterval(interval)
    }, [])

    return { jobs, loading, error, refetch: fetch }
}