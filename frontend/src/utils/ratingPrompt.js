// when to ask for a rating after a title is completed, kept apart so it can be tested

// a title completed this recently (in the player or a reader, say) is still asked about when
// its details page opens
export const RECENT_COMPLETION_MS = 24 * 60 * 60 * 1000

/** The localStorage key remembering "Not now" for one completion of one entry */
export function dismissKey(entry) {
    return `ratingPromptDismissed:${entry.id}:${entry.updated_at}`
}

/**
 * Whether to ask the user to rate a title
 * @param {string|null|undefined} previousStatus the status the page last saw; undefined when it
 *  hasn't seen the entry yet, null when the title wasn't on the list
 * @param {object} entry the user's list entry now, if any
 * @param {number} now the time now, in ms
 * @param {boolean} dismissed the user said "Not now" to this completion already
 */
export function shouldPromptRating(previousStatus, entry, now, dismissed) {
    if (!entry || entry.status !== "completed" || entry.rating != null || dismissed) return false
    // completed while the page was open
    if (previousStatus !== undefined) return previousStatus !== "completed"
    // completed somewhere else, a little while ago
    const updated = Date.parse(entry.updated_at)
    return Number.isFinite(updated) && now - updated < RECENT_COMPLETION_MS
}

const WORDS = ["", "Appalling", "Horrible", "Very bad", "Bad", "Average", "Fine", "Good", "Very good", "Great", "Masterpiece"]

/** A word for a rating from 1 to 10, empty for no rating */
export function ratingWord(rating) {
    return WORDS[rating] ?? ""
}
