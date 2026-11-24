/**
 * What a title in a merge stood for: one season of the series, its specials, or the series itself
 * @param {object} title a title from /admin/anime-merges: season (-1 for the series), sonarr
 */
export function seasonLabel(title) {
    if (title.sonarr || title.season < 0) return "series"
    return title.season === 0 ? "specials" : `season ${title.season}`
}

/**
 * The season typed when attaching a title to a series: a whole number, 0 for specials
 * @param {string} text what was typed
 * @returns {number|null} the season, null when it isn't one
 */
export function parseSeason(text) {
    const t = String(text ?? "").trim()
    if (!/^\d{1,3}$/.test(t)) return null
    return Number(t)
}

/**
 * The series a title can be attached to whose names match what was typed, without the title itself
 * @param {object[]} series from /admin/anime-series: id, title, tvdb_id
 * @param {string} query what was typed
 * @param {string} selfId the title being attached
 */
export function matchSeries(series, query, selfId) {
    const q = String(query ?? "").trim().toLowerCase()
    return (series ?? []).filter(s => s.id !== selfId && (!q || s.title.toLowerCase().includes(q)))
}
