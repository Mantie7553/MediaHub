/**
 * Read a remembered UI preference from localStorage
 * @param {string} key the storage key
 * @param {string} fallback returned when nothing is stored or storage is unavailable
 * @param {string[]} allowed when given, a stored value outside this list counts as nothing stored
 * @returns the stored string, or the fallback
 */
export function readStored(key, fallback, allowed) {
    try {
        const value = localStorage.getItem(key)
        if (value === null) return fallback
        if (allowed && !allowed.includes(value)) return fallback
        return value
    } catch {
        // private windows and blocked site data can refuse storage
        return fallback
    }
}

/**
 * Remember a UI preference in localStorage, silently doing nothing when storage is unavailable
 * @param {string} key the storage key
 * @param {string} value the value to store
 */
export function writeStored(key, value) {
    try {
        localStorage.setItem(key, value)
    } catch {
        // the preference just isn't remembered for next time
    }
}
