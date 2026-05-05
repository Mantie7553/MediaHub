import { useState, useEffect } from 'react'
import api from '../services/api'

/**
 * Hook for getting download requests from the database
 *  that sets the loading and error states
 * @returns a list of requests, the loading state, the error state, and a function to refetch the data
 */
export default function useRequests() {
    const [requests, setRequests] = useState([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState(null)

    function fetch() {
        setLoading(true)
        api.get('/requests')
            .then(res => setRequests(res.data))
            .catch(err => setError(err))
            .finally(() => setLoading(false))
    }

    useEffect(() => { fetch() }, [])

    return { requests, loading, error, refetch: fetch }
}