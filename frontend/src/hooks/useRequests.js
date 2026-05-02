import { useState, useEffect } from 'react'
import api from '../services/api'

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