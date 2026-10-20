/**
 * The options that contain the typed text, ignoring case: ones that start with it first, then
 * the rest, each group in the given order. An exact match isn't suggested, it's already typed
 * @param {string[]} options everything that could be suggested
 * @param {string} value what has been typed
 * @param {number} max the most suggestions to return
 */
export function suggestions(options, value, max = 8) {
    const q = value.trim().toLowerCase()
    const starts = []
    const contains = []
    for (const option of options) {
        const o = option.toLowerCase()
        if (o === q) continue
        if (o.startsWith(q)) starts.push(option)
        else if (o.includes(q)) contains.push(option)
    }
    return [...starts, ...contains].slice(0, max)
}

/**
 * The library's spelling of a name typed with different case or spacing, so "yoasobi" files
 * the track with "YOASOBI" instead of starting a second folder. Anything new is kept as typed,
 * trimmed
 * @param {string[]} options the names already in the library
 * @param {string} value what was typed
 */
export function canonical(options, value) {
    const v = value.trim()
    const key = v.replace(/\s+/g, " ").toLowerCase()
    return options.find(o => o.replace(/\s+/g, " ").toLowerCase() === key) ?? v
}

/**
 * An artist name from a YouTube channel name: auto-generated music channels are called
 * "Artist - Topic"
 * @param {string} uploader the channel name
 */
export function artistFromUploader(uploader) {
    return (uploader ?? "").replace(/\s+-\s+Topic$/i, "").trim()
}
