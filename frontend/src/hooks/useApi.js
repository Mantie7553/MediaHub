import { useCallback, useEffect, useRef, useState } from 'react'
import api from '../services/api'
import { isNotFound } from '../utils/errors'

/**
 * Load data from the API with loading, error and not-found states
 * @param {string|null} path the API path, or null to skip loading
 * @param {object} options
 *  - initial: the data before the first response, and when the response is empty
 *  - poll: milliseconds between background refreshes, 0 for none
 * @returns data, loading (until this path has answered), error (the axios error), notFound,
 *  pollError (a background refresh failed while the older data is still shown), and
 *  reload(background = false)
 */
export default function useApi(path, { initial = null, poll = 0 } = {}) {
    // bumped by reload(): a new attempt shows the loading state again
    const [attempt, setAttempt] = useState(0)
    const key = path === null ? null : `${attempt}:${path}`
    // the last answer and which attempt it belongs to. Loading is derived from it, so a new
    // path starts loading in the same render instead of showing the old page first
    const [result, setResult] = useState({ key: null, data: initial, error: null, pollError: null })
    const keyRef = useRef(key)
    const initialRef = useRef(initial)

    const fetchInto = useCallback((forKey, background) => {
        return api.get(path)
            .then(res => {
                if (keyRef.current !== forKey) return
                setResult({ key: forKey, data: res.data ?? initialRef.current, error: null, pollError: null })
            })
            .catch(err => {
                if (keyRef.current !== forKey) return
                // a failed refresh keeps what is on screen instead of replacing the page
                setResult(prev => background && prev.key === forKey && !prev.error
                    ? { ...prev, pollError: err }
                    : { key: forKey, data: initialRef.current, error: err, pollError: null })
            })
    }, [path])

    useEffect(() => {
        keyRef.current = key
        if (key === null) return
        fetchInto(key, false)
        const interval = poll ? setInterval(() => fetchInto(key, true), poll) : null
        return () => {
            if (interval) clearInterval(interval)
        }
    }, [key, poll, fetchInto])

    const reload = useCallback((background = false) => {
        if (keyRef.current === null) return Promise.resolve()
        if (background) return fetchInto(keyRef.current, true)
        setAttempt(a => a + 1)
        return Promise.resolve()
    }, [fetchInto])

    const loading = key !== null && result.key !== key
    const error = loading ? null : result.error

    return {
        data: result.data,
        loading,
        error,
        notFound: isNotFound(error),
        pollError: loading ? null : result.pollError,
        reload,
    }
}
