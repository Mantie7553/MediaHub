/*
Reads a chapter number out of a chapter file's name, to show it before uploading. The
server reads names the same way (backend/internal/mangafiles), and the number typed in the
upload dialog wins over both
*/

const bracketed = /^\[(\d+(?:\.\d+)?)\]/
const marked = /(?:^|[^a-z])(?:chapter|chap|ch|c)\.?\s*(\d+(?:\.\d+)?)|#(\d+(?:\.\d+)?)/i
const volume = /(?:^|[^a-z])(?:volume|vol|v)\.?\s*\d+/gi
const number = /\d+(?:\.\d+)?/g
const tags = /\[[^\]]*\]|\([^)]*\)|\{[^}]*\}/g

/**
 * The chapter number in a file name: "Chainsaw Man c098.cbz" -> "98"
 * @param {string} fileName the file's name
 * @param {string} series the series title, removed first so a number in it isn't read
 * @returns {string|null} the number without leading zeros, null when the name has none
 *  (like a whole volume, "Chainsaw Man v12.cbz")
 */
export function chapterNumberFromName(fileName, series = "") {
    let name = fileName.replace(/\.(cbz|zip)$/i, "")

    const b = name.match(bracketed)
    if (b) return clean(b[1])

    name = name.replace(tags, " ").replace(/_/g, " ")
    if (series) {
        const i = name.toLowerCase().indexOf(series.toLowerCase())
        if (i >= 0) name = name.slice(0, i) + " " + name.slice(i + series.length)
    }

    const m = name.match(marked)
    if (m) return clean(m[1] ?? m[2])

    // without a chapter marker, a volume marker means the file is a whole volume
    const rest = name.replace(volume, " ")
    const numbers = rest.match(number)
    if (rest !== name && !numbers) return null
    return numbers ? clean(numbers[numbers.length - 1]) : null
}

// "098" -> "98", "052.5" -> "52.5"
function clean(text) {
    const [whole, frac] = text.split(".")
    const w = whole.replace(/^0+(?=\d)/, "")
    return frac !== undefined ? `${w}.${frac}` : w
}

/**
 * Whether a file is a chapter archive the server accepts
 * @param {string} fileName the file's name
 */
export function isChapterArchive(fileName) {
    return /\.(cbz|zip)$/i.test(fileName)
}
