import { useState, useEffect, useRef, useCallback } from 'react'
import api from '../services/api'

// pages loaded ahead of the one on screen, so turning a page is usually instant
const PRELOAD = 2

/**
 * Custom hook for getting the current page the reader will display, and preloading the next ones
 * Handles the loading and error states
 * @param {*} id the manga id
 * @param {*} chapterId the chapter id
 * @param {number} initialPage the page to open on
 * @param {number} pageCount pages in the chapter, 0 when unknown
 * @returns the current page and its setter, the page image, loading, error (the axios error),
 *  and refetch to try the current page again
 */
export default function usePages(id, chapterId, initialPage = 0, pageCount = 0) {
    const [currentPage, setCurrentPage] = useState(initialPage);
    // page number -> object URL, and page number -> error, for this chapter
    const [pages, setPages] = useState({});
    const [errors, setErrors] = useState({});
    const [attempt, setAttempt] = useState(0);
    // pages requested so far (loaded or in flight), so preloading never fetches one twice
    const requestedRef = useRef(new Set());
    const urlsRef = useRef([]);

    useEffect(() => {
        let cancelled = false;
        const requested = requestedRef.current;
        const wanted = [];
        for (let page = currentPage; page <= currentPage + PRELOAD; page++) {
            if (page >= 0 && (!pageCount || page < pageCount)) wanted.push(page);
        }

        // the current page first, then the next ones in order. A page that was already in
        // flight when the reader moved on still lands in `pages` when it arrives
        async function load() {
            for (const page of wanted) {
                if (cancelled) return;
                if (requested.has(page)) continue;
                requested.add(page);
                // a page that failed while preloading gets a fresh try here
                setErrors(prev => {
                    if (!(page in prev)) return prev;
                    const next = { ...prev };
                    delete next[page];
                    return next;
                });
                try {
                    const resp = await api.get(`/manga/${id}/chapters/${chapterId}/pages/${page}`, { responseType: 'blob' });
                    const objectUrl = URL.createObjectURL(resp.data);
                    urlsRef.current.push(objectUrl);
                    setPages(prev => ({ ...prev, [page]: objectUrl }));
                } catch (err) {
                    requested.delete(page);
                    setErrors(prev => ({ ...prev, [page]: err }));
                    return;
                }
            }
        }
        load();

        return () => { cancelled = true };
    }, [id, chapterId, currentPage, pageCount, attempt])

    // free the images when leaving the chapter
    useEffect(() => {
        const urls = urlsRef.current;
        return () => urls.forEach(url => URL.revokeObjectURL(url));
    }, [])

    const refetch = useCallback(() => {
        setErrors(prev => {
            const next = { ...prev };
            delete next[currentPage];
            return next;
        });
        setAttempt(a => a + 1);
    }, [currentPage]);

    const imageSrc = pages[currentPage] ?? null;
    const error = imageSrc ? null : errors[currentPage] ?? null;

    return {currentPage, setCurrentPage, imageSrc, loading: !imageSrc && !error, error, refetch}
}
