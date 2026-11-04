/**
 * Where the skip segments sit on the seek bar, as percentages of its width
 * @param {object[]} segments [{ type, start, end }] in seconds
 * @param {number} duration the video's length in seconds
 * @returns [{ type, left, width }], none while the length isn't known
 */
export function segmentMarks(segments, duration) {
    if (!(duration > 0) || !Number.isFinite(duration)) return []
    const clamp = t => Math.min(Math.max(t, 0), duration)
    return segments
        .map(seg => ({ type: seg.type, start: clamp(seg.start), end: clamp(seg.end) }))
        .filter(seg => seg.end > seg.start)
        .map(seg => ({
            type: seg.type,
            left: (seg.start / duration) * 100,
            width: ((seg.end - seg.start) / duration) * 100,
        }))
}

/**
 * How far ahead the video has loaded from where it's playing
 * @param {TimeRanges} ranges the video's buffered ranges
 * @param {number} t where it's playing, in seconds
 * @returns the end of the loaded range t is in, 0 when t isn't loaded
 */
export function bufferedEnd(ranges, t) {
    // a range starting just ahead still counts, the video sits a moment before it after a seek
    for (let i = 0; i < (ranges?.length ?? 0); i++) {
        if (ranges.start(i) <= t + 0.5 && t <= ranges.end(i)) return ranges.end(i)
    }
    return 0
}

/**
 * How far along the seek bar a pointer is
 * @param {number} clientX the pointer's x
 * @param {{ left: number, width: number }} rect the bar's bounding box
 * @returns 0 to 1
 */
export function seekFraction(clientX, rect) {
    if (!(rect.width > 0)) return 0
    return Math.min(Math.max((clientX - rect.left) / rect.width, 0), 1)
}

/**
 * Where the picture of an object-contain video sits in its box
 * @param {{ width: number, height: number }} box the video element's size
 * @param {number} videoWidth the picture's own width, 0 before it's known
 * @param {number} videoHeight the picture's own height
 * @returns { left, top, width, height }, the whole box until the picture's size is known
 */
export function pictureRect(box, videoWidth, videoHeight) {
    if (!(videoWidth > 0 && videoHeight > 0 && box.width > 0 && box.height > 0)) {
        return { left: 0, top: 0, width: box.width, height: box.height }
    }
    const scale = Math.min(box.width / videoWidth, box.height / videoHeight)
    const width = videoWidth * scale
    const height = videoHeight * scale
    return { left: (box.width - width) / 2, top: (box.height - height) / 2, width, height }
}

/**
 * How far the subtitles move up to stay clear of the bottom controls: as far as the controls
 * reach into the picture, nothing when the black band under the picture fits them
 * @param {number} barHeight the bottom controls' height
 * @param {{ width: number, height: number }} box the video element's size
 * @param {number} videoWidth the picture's own width
 * @param {number} videoHeight the picture's own height
 * @returns pixels to move them up
 */
export function subtitleLift(barHeight, box, videoWidth, videoHeight) {
    const picture = pictureRect(box, videoWidth, videoHeight)
    const below = box.height - picture.top - picture.height
    return Math.max(0, barHeight - below)
}

// the WebVTT tags kept for styling, everything else (classes, voices, timestamps) is dropped
const KEPT_TAGS = /^<(\/?)(i|b|u)(?:\.[^>]*)?>$/
const ENTITY = /&(?!(?:amp|lt|gt|nbsp|lrm|rlm|quot|#\d+|#x[\da-f]+);)/gi

/**
 * A WebVTT cue's text as HTML to show: italics, bold and underline, nothing else
 * @param {string} text the cue's text
 * @returns the HTML
 */
export function cueHtml(text) {
    return (text ?? "")
        .split(/(<[^>]*>)/)
        .map(part => {
            if (part.startsWith("<")) {
                const tag = part.match(KEPT_TAGS)
                return tag ? `<${tag[1]}${tag[2]}>` : ""
            }
            return part.replace(ENTITY, "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll("\n", "<br>")
        })
        .join("")
}
