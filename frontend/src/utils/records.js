/**
 * How many records have each status, in the given order, skipping statuses nobody has.
 * Statuses missing from the order come last, so a new one still gets a tab
 * @param {object[]} items the records, each with a status
 * @param {string[]} order the statuses in the order to show them
 * @returns [{ key, count }]
 */
export function statusCounts(items, order) {
    const counts = new Map()
    for (const item of items) counts.set(item.status, (counts.get(item.status) ?? 0) + 1)
    const known = order.filter(s => counts.has(s))
    const extra = [...counts.keys()].filter(s => !order.includes(s))
    return [...known, ...extra].map(key => ({ key, count: counts.get(key) }))
}

/**
 * The records with a status ("all" for any) whose title contains the query, ignoring case
 * @param {object[]} items the records
 * @param {string} status the status to keep, or "all"
 * @param {string} query text to look for in the title, empty for any
 * @param {function} titleOf reads a record's title
 */
export function filterRecords(items, status, query, titleOf) {
    const q = query.trim().toLowerCase()
    return items.filter(item =>
        (status === "all" || item.status === status) &&
        (!q || (titleOf(item) ?? "").toLowerCase().includes(q)))
}
