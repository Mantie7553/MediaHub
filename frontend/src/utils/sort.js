// sorting lists in the browser. Every list arrives whole, so a sort is just a comparator

// numbers inside names sort as numbers ("Vol 2" before "Vol 10"), and case and accents don't count
const collator = new Intl.Collator(undefined, { numeric: true, sensitivity: "base" })

const isMissing = value => value === null || value === undefined || value === "" || Number.isNaN(value)

/**
 * Compare two values of the same kind. Missing ones always go last, whichever way the sort runs,
 * so an unrated title never tops "Lowest rated"
 * @param {any} a a string, number or timestamp
 * @param {any} b the other value
 * @param {boolean} desc largest (or Z) first
 */
export function compareValues(a, b, desc = false) {
    const aMissing = isMissing(a)
    const bMissing = isMissing(b)
    if (aMissing || bMissing) return aMissing === bMissing ? 0 : aMissing ? 1 : -1
    const result = typeof a === "string" && typeof b === "string"
        ? collator.compare(a, b)
        : a < b ? -1 : a > b ? 1 : 0
    return desc ? -result : result
}

/** A comparator on text a getter picks out, A to Z (or Z to A with desc) */
export const text = (get, desc = false) => (a, b, ctx) =>
    compareValues(get(a, ctx) ?? null, get(b, ctx) ?? null, desc)

/** A comparator on a number a getter picks out, smallest first (or largest with desc) */
export const number = (get, desc = false) => (a, b, ctx) =>
    compareValues(get(a, ctx) ?? null, get(b, ctx) ?? null, desc)

// a date string or Date as a timestamp, or null
const timestamp = value => {
    if (value === null || value === undefined || value === "") return null
    const time = new Date(value).getTime()
    return Number.isNaN(time) ? null : time
}

/** A comparator on a date a getter picks out, oldest first (or newest with desc) */
export const date = (get, desc = false) => (a, b, ctx) =>
    compareValues(timestamp(get(a, ctx)), timestamp(get(b, ctx)), desc)

/** Comparators tried in turn, each breaking the ties the one before leaves */
export const thenBy = (...comparators) => (a, b, ctx) => {
    for (const compare of comparators) {
        const result = compare(a, b, ctx)
        if (result !== 0) return result
    }
    return 0
}

/**
 * A sorted copy of a list; the list itself is left alone. An option without a comparator keeps
 * the order the list came in, like a playlist's own order
 * @param {any[]} items the list
 * @param {object} option a sort option, { key, label, compare(a, b, ctx) }
 * @param {object} ctx passed to the comparator, e.g. { userContentMap }
 */
export function sortItems(items, option, ctx = {}) {
    if (!items || !option?.compare) return items
    return [...items].sort((a, b) => option.compare(a, b, ctx))
}

/**
 * The option a stored key names, or the first (the default) when it names none of them, like a
 * sort that was renamed or doesn't apply to this list
 * @param {object[]} options a list's sort options
 * @param {string} key the stored key
 */
export function pickOption(options, key) {
    return options.find(option => option.key === key) ?? options[0]
}

// list entries name their title media_title, library items title, collections name
export const titleOf = item => item.media_title ?? item.title ?? item.name ?? ""

/** Title A to Z, the order lists used before they could be sorted */
export const byTitle = (a, b) => compareValues(titleOf(a), titleOf(b))
