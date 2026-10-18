import { useState, useEffect, useCallback } from 'react'
import api from '../services/api'

/**
 * Custom hook for getting the current page the reader will display
 * Handles the loading and error states
 * @param {*} id the manga id
 * @param {*} chapterId the chapter id
 * @param {number} initialPage the page to open on
 * @returns the current page and its setter, the page image, loading, error (the axios error),
 *  and refetch to try the current page again
 */
export default function usePages(id, chapterId, initialPage = 0) {
    const [currentPage, setCurrentPage] = useState(initialPage);
    const [attempt, setAttempt] = useState(0);
    const key = `${id}/${chapterId}/${currentPage}/${attempt}`;
    // the last loaded page and which request it belongs to, loading is derived from it
    const [result, setResult] = useState({ key: null, src: null, error: null });

    useEffect(() => {
        let objectUrl = null;
        let cancelled = false;
        api.get(`/manga/${id}/chapters/${chapterId}/pages/${currentPage}`, { responseType: 'blob' })
            .then(resp => {
                if (cancelled) return;
                objectUrl = URL.createObjectURL(resp.data);
                setResult({ key, src: objectUrl, error: null });
            })
            .catch(err => { if (!cancelled) setResult({ key, src: null, error: err }) });

        return () => {
            cancelled = true;
            if (objectUrl) URL.revokeObjectURL(objectUrl);
        }
    }, [id, chapterId, currentPage, key])

    const refetch = useCallback(() => setAttempt(a => a + 1), []);
    const loading = result.key !== key;

    return {currentPage, setCurrentPage, imageSrc: result.src, loading, error: loading ? null : result.error, refetch}
}
