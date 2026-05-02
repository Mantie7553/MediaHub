import { useState, useEffect } from 'react'
import api from '../services/api'

export default function useJobs() {
    const [jobs, setJobs] = useState([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState(null)

    useEffect(() => {
        api.get('/admin/jobs')
            .then(res => setJobs(res.data))
            .catch(err => setError(err))
            .finally(() => setLoading(false))
    }, [])

    return { jobs, loading, error }
}