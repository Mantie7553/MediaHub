import { useCallback, useEffect, useRef, useState } from 'react'
import api from '../services/api'
import { forgetCached, getCached, setCached } from '../services/apiCache'
import { isNotFound } from '../utils/errors'

/**
 * Load data from the API with loading, error and not-found states. A path loaded before in this
 * tab shows its last data straight away and is refreshed in the background, so going back to a
 * page is instant
 * @param {string|null} path the API path, or null to skip loading
 * @param {object} options
 *  - initial: the data before the first response, and when the response is empty
 *  - poll: milliseconds between background refreshes, 0 for none
 * @returns data, loading (until this path has answered, unless it has data from before), error
 *  (the axios error), notFound, pollError (a background refresh failed while the older data is
 *  still shown), and reload(background = false)
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
                const data = res.data ?? initialRef.current
                setCached(path, data)
                setResult({ key: forKey, data, error: null, pollError: null })
            })
            .catch(err => {
                if (keyRef.current !== forKey) return
                const cached = getCached(path)
                // a failed refresh keeps what is on screen instead of replacing the page, unless
                // the thing itself is gone
                setResult(prev => {
                    const shown = prev.key === forKey ? (prev.error ? undefined : prev.data) : cached
                    if (background && shown !== undefined && !isNotFound(err)) {
                        return { key: forKey, data: shown, error: null, pollError: err }
                    }
                    return { key: forKey, data: initialRef.current, error: err, pollError: null }
                })
                if (isNotFound(err) || !background) forgetCached(path)
            })
    }, [path])

    useEffect(() => {
        keyRef.current = key
        if (key === null) return
        // with data from before on screen, this load is a background refresh
        fetchInto(key, getCached(path) !== undefined)
        const interval = poll ? setInterval(() => fetchInto(key, true), poll) : null
        return () => {
            if (interval) clearInterval(interval)
        }
    }, [key, path, poll, fetchInto])

    const reload = useCallback((background = false) => {
        if (keyRef.current === null) return Promise.resolve()
        if (background) return fetchInto(keyRef.current, true)
        forgetCached(path)
        setAttempt(a => a + 1)
        return Promise.resolve()
    }, [path, fetchInto])

    const pending = key !== null && result.key !== key
    const cached = pending ? getCached(path) : undefined
    const loading = pending && cached === undefined
    const error = pending ? null : result.error

    return {
        data: pending && cached !== undefined ? cached : result.data,
        loading,
        error,
        notFound: isNotFound(error),
        pollError: pending ? null : result.pollError,
        reload,
    }
}
