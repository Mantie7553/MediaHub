import { useState, useEffect } from 'react'
import api from '../services/api'

export default function useMediaItem(id) {
    const [item, setItem] = useState(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState("");

    function fetch(showLoading = true) {
        if (showLoading) setLoading(true)
        api.get(`/media/${id}`)
        .then(resp => setItem(resp.data))
        .catch(err => setError(err.message))
        .finally(() => setLoading(false))
    }

    useEffect(() => {
        fetch(true);
    }, [])

    return { item, loading, error, refetch: fetch }
}