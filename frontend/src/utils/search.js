// lower case without accents, so "pokemon" finds "Pokémon"
function fold(text) {
    return (text ?? "").normalize("NFD").replace(/\p{Diacritic}/gu, "").toLowerCase()
}

/**
 * Whether an item matches what was typed: every word has to appear in its title (a playlist or
 * collection's name) or artist
 * @param {object} item a title, list entry or album
 * @param {string} query what was typed
 */
export function matchesQuery(item, query) {
    const words = fold(query).split(/\s+/).filter(Boolean)
    if (words.length === 0) return true
    const text = fold(`${item.media_title ?? item.title ?? item.name ?? ""} ${item.artist ?? ""}`)
    return words.every(word => text.includes(word))
}

/**
 * The items that match what was typed, in their order
 * @param {object[]} items titles, list entries or albums
 * @param {string} query what was typed
 */
export function filterByQuery(items, query) {
    return items.filter(item => matchesQuery(item, query))
}

// how far the page has to move before the search bar hides or comes back, so a shaky scroll
// doesn't flicker it, and how near the top it always shows
export const SCROLL_STEP = 8
export const ALWAYS_SHOWN_ABOVE = 120

/**
 * Whether a search bar that hides on scroll should be hidden after the page moved
 * @param {boolean} hidden whether it's hidden now
 * @param {number} fromY where the page was when it last changed
 * @param {number} toY where the page is now
 * @returns hidden, and whether toY becomes the new starting point
 */
export function hideOnScroll(hidden, fromY, toY) {
    if (toY <= ALWAYS_SHOWN_ABOVE) return { hidden: false, moved: true }
    const delta = toY - fromY
    if (Math.abs(delta) < SCROLL_STEP) return { hidden, moved: false }
    return { hidden: delta > 0, moved: true }
}
