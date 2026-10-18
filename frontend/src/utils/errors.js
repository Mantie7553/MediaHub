/**
 * Turn an API error into one sentence a user can act on. The backend sends errors as
 * plain text (http.Error), not JSON, so the response body is usually the message itself
 * @param {any} err an axios error, or anything thrown
 * @param {string} fallback what to say when nothing more specific is known
 * @returns a readable message
 */
export function errorMessage(err, fallback = "Something went wrong") {
    if (typeof err === "string") return err || fallback
    if (!err) return fallback

    // axios sets request but no response when the server could not be reached at all
    if (err.request && !err.response) {
        return "Can't reach the server. Check your connection and try again"
    }

    const status = err.response?.status
    if (status === 502 || status === 503 || status === 504) {
        return "The server is restarting or unreachable, try again in a moment"
    }
    if (status === 404) return "Not found"

    const data = err.response?.data
    if (typeof data === "string" && data.trim()) return capitalize(data.trim())
    if (typeof data?.error === "string" && data.error) return capitalize(data.error)

    return fallback
}

/**
 * Whether an error means the thing asked for does not exist
 * @param {any} err an axios error
 */
export function isNotFound(err) {
    return err?.response?.status === 404
}

function capitalize(text) {
    return text.charAt(0).toUpperCase() + text.slice(1)
}
