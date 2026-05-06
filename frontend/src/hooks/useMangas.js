import { useState, useEffect } from 'react'
import api from '../services/api'

/**
 * Custom hook for getting information for a specific manga
 * @returns manga: the current manga object, totalPages: total number of pages for the current chapter,
 *  loading: if the page is loading or not, error: the error message to be displayed, refetch: function for getting the
 * data again
 */
export default function usePages() {
    const [manga, setManga] = useState([]);
    const [totalPages, setTotalPages] = useState(0);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    function fetch(showLoading = true) {
        api.get(`/media/${id}`)
        .then(resp => {
            setManga(resp.data);
            const match = resp.data.metadata.chapters?.find(c => c.id === chapterId);
            if (match) setTotalPages(match.page_count);

        })
        .catch(err => setError(err.message))
        .finally(() => setLoading(false))
    }

    useEffect(() => {
        fetch(true);
    }, [currentPage])

    return { manga, totalPages, loading, error, refetch: fetch }
}