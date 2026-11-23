// the build this tab is running; set by vite.config.js, missing in tests
export const CURRENT_BUILD = import.meta.env.VITE_BUILD_ID ?? null

/**
 * The build the server has now, from /version.json
 * @returns {Promise<string|null>} its ID, or null when it can't be told (offline, or the dev server)
 */
export async function fetchBuildId() {
    try {
        const res = await fetch("/version.json", { cache: "no-store" })
        if (!res.ok) return null
        const data = await res.json()
        return typeof data?.build === "string" ? data.build : null
    } catch {
        return null
    }
}

/**
 * Whether the server has a different build than this tab is running. Unknown on either side is
 * never new, so a failed check doesn't reload
 * @param {string|null} current
 * @param {string|null} latest
 */
export function isNewBuild(current, latest) {
    return Boolean(current && latest && current !== latest)
}
