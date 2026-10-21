/*
Helpers for the admin editing dialogs: reading episode, volume and track numbers out of file
names to fill in an upload's details, and turning a title's saved values into form values and
back
*/

/**
 * Season and episode from a video file's name: "Frieren - S01E05 - Title.mkv",
 * "[Group] Frieren - 05 [1080p].mkv", "Episode 5.mp4"
 * @param {string} fileName the file's name
 * @returns {{season: string, episode: string}} empty strings for what isn't in the name
 */
export function episodeFromName(fileName) {
    const name = fileName.replace(/\.[^.]+$/, "")
    const sxe = name.match(/s(\d{1,2})\s*e(\d{1,4})/i) ?? name.match(/\b(\d{1,2})x(\d{2,4})\b/)
    if (sxe) return { season: trimZeros(sxe[1]), episode: trimZeros(sxe[2]) }

    const marked = name.match(/(?:episode|ep|e)\.?\s*(\d{1,4})\b/i)
    if (marked) return { season: "", episode: trimZeros(marked[1]) }

    // fansub style, "Title - 05", ignoring resolutions and tags
    const plain = name.replace(/\[[^\]]*\]|\([^)]*\)/g, " ").match(/[-–]\s*(\d{1,4})(?:v\d)?\s*$/)
    return { season: "", episode: plain ? trimZeros(plain[1]) : "" }
}

/**
 * The volume number in a light novel file's name: "Mushoku Tensei Volume 3.epub", "v03", "Vol. 3"
 * @param {string} fileName the file's name
 * @returns {string} the number, or "" when there is none
 */
export function volumeFromName(fileName) {
    const m = fileName.replace(/\.epub$/i, "").match(/(?:volume|vol|v)[._\s]*(\d{1,4})/i)
    return m ? trimZeros(m[1]) : ""
}

/**
 * A track's number and title from its file's name: "03 - Song.mp3", "03. Song.mp3", "Song.mp3"
 * @param {string} fileName the file's name
 * @returns {{trackNumber: string, title: string}}
 */
export function trackFromName(fileName) {
    const name = fileName.replace(/\.mp3$/i, "").replace(/_/g, " ").trim()
    const m = name.match(/^(\d{1,3})\s*(?:[-.–)]\s*|\s+)(.+)$/)
    if (m) return { trackNumber: trimZeros(m[1]), title: m[2].trim() }
    return { trackNumber: "", title: name }
}

function trimZeros(digits) {
    return digits.replace(/^0+(?=\d)/, "")
}

/**
 * A description as it is edited: descriptions from AniList are HTML with <br> line breaks,
 * shown in the text box as new lines
 * @param {string|null} description as saved
 * @param {boolean} html whether the page shows it as HTML
 */
export function descriptionToText(description, html) {
    if (!description) return ""
    if (!html) return description
    return description.replace(/\r?\n/g, "").replace(/<br\s*\/?>/gi, "\n")
}

/**
 * The description to save from the text box, the reverse of descriptionToText
 * @param {string} text what was typed
 * @param {boolean} html whether the page shows it as HTML
 * @returns {string|null} null when empty, which clears it
 */
export function textToDescription(text, html) {
    const trimmed = text.trim()
    if (!trimmed) return null
    return html ? trimmed.replace(/\r?\n/g, "<br>") : trimmed
}

/**
 * Genres typed as a comma separated list
 * @param {string} text "Action, Fantasy"
 * @returns {string[]} trimmed, without empty or repeated ones
 */
export function genresFromText(text) {
    const seen = new Set()
    return text.split(",").map(g => g.trim()).filter(g => {
        const key = g.toLowerCase()
        if (!g || seen.has(key)) return false
        seen.add(key)
        return true
    })
}

/**
 * Only the fields that changed, ready to send. Empty strings become null (cleared) and
 * number fields become numbers
 * @param {object} initial the form's starting values
 * @param {object} values the form's values now
 * @param {string[]} numberFields fields sent as numbers
 */
export function changedFields(initial, values, numberFields = []) {
    const out = {}
    for (const [key, value] of Object.entries(values)) {
        if (JSON.stringify(value) === JSON.stringify(initial[key])) continue
        if (typeof value === "string" && value.trim() === "") out[key] = null
        else if (numberFields.includes(key)) out[key] = Number(value)
        else out[key] = typeof value === "string" ? value.trim() : value
    }
    return out
}

/**
 * A YYYY-MM-DD date for an <input type="date"> from a saved timestamp
 * @param {string|null} value "2023-09-29T00:00:00Z"
 */
export function dateInputValue(value) {
    if (!value) return ""
    const date = String(value).slice(0, 10)
    // a year of 0001 means the date was never known
    return /^\d{4}-\d{2}-\d{2}$/.test(date) && !date.startsWith("0001") ? date : ""
}
