import { useState, useEffect } from 'react'
import api from '../services/api'

export default function usePages() {
    const [currentPage, setCurrentPage] = useState(0);
    const [imageSrc, setImageSrc] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    function fetch(showLoading = true) {
        if (showLoading) setLoading(true)
        let objectUrl = null;
        api.get(`/manga/${id}/chapters/${chapterId}/pages/${currentPage}`, { responseType: 'blob' })
        .then(resp => {
            objectUrl = URL.createObjectURL(resp.data);
            setImageSrc(objectUrl);
        })
        .catch(err => setError(err.message))
        .finally(() => setLoading(false));
        return () => { if (objectUrl) URL.revokeObjectURL(objectUrl) }
    }

    useEffect(() => {
        fetch(true);
    }, [currentPage])

    return {currentPage, imageSrc, loading, error, refetch: fetch}
}