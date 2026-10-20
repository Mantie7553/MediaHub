// most API responses remembered at once, the least recently loaded dropped first
const MAX_ENTRIES = 100

// path -> the last successful response's data, oldest load first
const entries = new Map()

/**
 * The last data loaded for a path, so a page seen before can show it straight away while
 * it loads again. Lives in memory only, for this tab. Read while rendering, so it changes nothing
 * @param {string} path the API path
 * @returns the data, or undefined when this path hasn't loaded yet
 */
export function getCached(path) {
    return entries.get(path)
}

/**
 * Remember the data a path answered with
 * @param {string} path the API path
 * @param {any} data the response data
 */
export function setCached(path, data) {
    entries.delete(path)
    entries.set(path, data)
    if (entries.size > MAX_ENTRIES) entries.delete(entries.keys().next().value)
}

/**
 * Forget one path, so its next load shows the loading state
 * @param {string} path the API path
 */
export function forgetCached(path) {
    entries.delete(path)
}

/**
 * Forget everything, when signing out so the next user never sees this one's data
 */
export function clearCached() {
    entries.clear()
}
