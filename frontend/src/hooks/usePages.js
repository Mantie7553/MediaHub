import { useState, useEffect } from 'react'
import api from '../services/api'

/**
 * Custom hook for getting the current page the reader will display
 * Handles the loading and error states
 * @param {*} id 
 * @param {*} chapterId 
 * @returns 
 */
export default function usePages(id, chapterId, initialPage = 0) {
    const [currentPage, setCurrentPage] = useState(initialPage);
    const [imageSrc, setImageSrc] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    let objectUrl = null;

    function fetch(showLoading = true) {
        if (showLoading) setLoading(true)
        api.get(`/manga/${id}/chapters/${chapterId}/pages/${currentPage}`, { responseType: 'blob' })
        .then(resp => {
            objectUrl = URL.createObjectURL(resp.data);
            setImageSrc(objectUrl);
        })
        .catch(err => setError(err.message))
        .finally(() => setLoading(false));
    }

    useEffect(() => {
        fetch(true);
        return () => { if (objectUrl) URL.revokeObjectURL(objectUrl) }
    }, [currentPage])

    return {currentPage, setCurrentPage, imageSrc, loading, error, refetch: fetch}
}