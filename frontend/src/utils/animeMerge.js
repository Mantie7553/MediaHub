/**
 * What a title in a merge stood for: one season of the series, its specials, or the series itself
 * @param {object} title a title from /admin/anime-merges: season (-1 for the series), sonarr
 */
export function seasonLabel(title) {
    if (title.sonarr || title.season < 0) return "series"
    return title.season === 0 ? "specials" : `season ${title.season}`
}
