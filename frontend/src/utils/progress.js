// episode, chapter and volume counts on the user's list, kept apart so they can be tested

// how high the progress wheel goes for a search result whose source doesn't say how long it is
export const OPEN_PROGRESS_MAX = 999

const PARTS = {
    anime: { label: "Episodes watched", short: "E" },
    manga: { label: "Chapters read", short: "Ch " },
    light_novel: { label: "Volumes read", short: "Vol " },
}

/** Whether a media type tracks progress as a count of parts */
export function hasParts(type) {
    return type in PARTS
}

/** The progress heading for a media type, like "Chapters read" */
export function partsLabel(type) {
    return PARTS[type]?.label ?? "Progress"
}

/**
 * How far the progress wheel goes: the title's length when its source knows it, otherwise the
 * files the library has (never below a count already given)
 * @param {object} info progress_count, progress_total and progress_total_known for a title in
 *  the library (its list entry, or /me/progress), if known
 * @param {object} item the title; a search result has total_parts when its source knows it
 * @returns { max, known }: the wheel's top, and whether that's the title's real length
 */
export function progressRange(info, item) {
    if (info?.progress_total != null) {
        return { max: Math.max(info.progress_total, info.progress_count ?? 0), known: !!info.progress_total_known }
    }
    const total = item?.total_parts ?? item?.total_chapters ?? item?.total_volumes
    return total > 0 ? { max: total, known: true } : { max: OPEN_PROGRESS_MAX, known: false }
}

/**
 * The progress line under a card's title, like "Ch 12 / 100", or "Ch 12" when the length isn't
 * known; null before the first part
 */
export function progressMeta(type, entry) {
    const part = PARTS[type]
    const count = entry?.progress_count ?? 0
    if (!part || count <= 0) return null
    const total = entry.progress_total_known ? entry.progress_total : null
    return `${part.short}${count}${total ? ` / ${total}` : ""}`
}

/** How far through a title, 0 to 100, for a card's bar; 0 when its length isn't known */
export function progressPercent(entry) {
    const total = entry?.progress_total_known ? entry.progress_total : 0
    if (!total) return 0
    return Math.min(100, Math.round(((entry.progress_count ?? 0) / total) * 100))
}
