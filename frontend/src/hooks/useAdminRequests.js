import { useState, useEffect } from 'react'
import api from '../services/api'

/**
 * Hook for getting all download requests from the database (admin only)
 * @returns a list of all requests, the loading state, the error state, and a function to refetch the data
 */
export default function useAdminRequests() {
    const [requests, setRequests] = useState([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState(null)

    function fetch() {
        setLoading(true)
        api.get('/requests/all')
            .then(res => setRequests(res.data))
            .catch(err => setError(err))
            .finally(() => setLoading(false))
    }

    useEffect(() => { fetch() }, [])

    return { requests, loading, error, refetch: fetch }
}