// where light novels are downloaded from by hand: the server can't fetch from it (it sits
// behind Cloudflare's browser check), so the app only links there
export const LIGHT_NOVEL_SITE = "https://server.elscione.com/"

/**
 * Opens the light novel site in a new tab with the title copied, to find in its folder listing
 * (it has no search to link to)
 * @param {string} title the light novel's title
 * @returns {Promise<boolean>} whether the title was copied
 */
export async function openNovelSite(title) {
    // opened first, while the click still counts as the user's, so it isn't blocked as a popup
    window.open(LIGHT_NOVEL_SITE, "_blank", "noopener,noreferrer")
    try {
        await navigator.clipboard.writeText(title)
        return true
    } catch {
        return false
    }
}
