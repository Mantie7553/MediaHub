import { BookOpen, BookText, Film, LayoutGrid, Music, Tv } from "lucide-react"

/**
 * The home pages: the overview of everything the user tracks, and one page per type with the
 * user's titles and the server's library of that type. The user picks one as their home, which
 * "/" opens
 *  - key: what the server stores as home_page
 *  - type: the media type the API filters on, null for the overview
 *  - discover: the Discover tab for this type, null for the overview
 */
export const homePages = [
    { key: "overview", type: null, path: "/home", label: "Overview", noun: "everything", icon: LayoutGrid, discover: null },
    { key: "anime", type: "anime", path: "/anime", label: "Anime", noun: "anime", icon: Tv, discover: "anime" },
    { key: "movie", type: "movie", path: "/movies", label: "Movies", noun: "movies", icon: Film, discover: "movie" },
    { key: "manga", type: "manga", path: "/manga", label: "Manga", noun: "manga", icon: BookOpen, discover: "manga" },
    { key: "music", type: "music_album", path: "/music", label: "Music", noun: "music", icon: Music, discover: "music" },
    { key: "light_novel", type: "light_novel", path: "/light-novels", label: "Light Novels", noun: "light novels", icon: BookText, discover: "light_novel" },
]

export const DEFAULT_HOME = "overview"

/**
 * The home page with this key, the overview for anything unknown (an old or missing setting)
 * @param {string} key a home_page value
 */
export function homePage(key) {
    return homePages.find(page => page.key === key) ?? homePages[0]
}

/**
 * Where a home page lives
 * @param {string} key a home_page value, unknown ones go to the overview
 */
export function homePath(key) {
    return homePage(key).path
}

/**
 * The home page for a Discover tab, so Discover can link to "your" side of the same type
 * @param {string} discoverKey the Discover tab's key (?type=)
 * @returns the home page, or undefined
 */
export function homePageForDiscover(discoverKey) {
    return homePages.find(page => page.discover === discoverKey)
}

/**
 * Whether a path is one of the home pages (or "/", which opens one), for highlighting Home in
 * the navigation. Details pages under the same prefix (/anime/12) are not
 * @param {string} pathname the current location's path
 */
export function isHomePath(pathname) {
    const path = pathname.length > 1 ? pathname.replace(/\/+$/, "") : pathname
    return path === "/" || homePages.some(page => page.path === path)
}
