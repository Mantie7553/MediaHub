import { useState, useEffect } from 'react'
import api from '../services/api'

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