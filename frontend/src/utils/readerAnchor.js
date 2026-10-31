/**
 * The API cache key for the last reading position the reader saved for a volume in this tab. The
 * title's cached data can be from before that save, so the reader opens from this when it's there
 * @param {string} volumeId the volume
 * @returns {string}
 */
export function savedPositionKey(volumeId) {
    return `/light-novels/volumes/${volumeId}/position`
}

// the elements a reading position or bookmark points at. A volume's markup is the same on every
// load, so the n-th of these is a stable address however tall the images above it turn out
export const BLOCK_SELECTOR = "p, h1, h2, h3, h4, h5, h6, li, blockquote, pre, img, hr, table"

/**
 * The blocks of a volume, in reading order
 * @param {Element} article the element the volume's markup is in
 * @returns {Element[]}
 */
export function getBlocks(article) {
    return Array.from(article.querySelectorAll(BLOCK_SELECTOR))
}

/**
 * How many characters of a block's text come before a point in it
 * @param {Element} block the block
 * @param {Node} node a text node in the block, or an element (then the point is its start)
 * @param {number} offset the offset in that text node
 * @returns {number}
 */
export function textOffsetIn(block, node, offset) {
    if (node.nodeType !== Node.TEXT_NODE) return 0
    const walker = block.ownerDocument.createTreeWalker(block, NodeFilter.SHOW_TEXT)
    let total = 0
    for (let n = walker.nextNode(); n; n = walker.nextNode()) {
        if (n === node) return total + offset
        total += n.length
    }
    return 0
}

/**
 * A range over the character at an offset in a block's text, the last character when the offset
 * runs past the end
 * @param {Element} block the block
 * @param {number} char the offset, as textOffsetIn counts it
 * @returns {Range|null} null when the block has no text
 */
export function rangeAt(block, char) {
    const walker = block.ownerDocument.createTreeWalker(block, NodeFilter.SHOW_TEXT)
    let last = null
    let total = 0
    for (let n = walker.nextNode(); n; n = walker.nextNode()) {
        if (n.length === 0) continue
        if (char < total + n.length) return charRange(n, char - total)
        total += n.length
        last = n
    }
    return last ? charRange(last, last.length - 1) : null
}

function charRange(node, i) {
    const range = node.ownerDocument.createRange()
    range.setStart(node, i)
    range.setEnd(node, i + 1)
    return range
}

// the text node and offset under a point, from whichever API the browser has
function caretAt(doc, x, y) {
    if (doc.caretPositionFromPoint) {
        const p = doc.caretPositionFromPoint(x, y)
        return p && { node: p.offsetNode, offset: p.offset }
    }
    if (doc.caretRangeFromPoint) {
        const r = doc.caretRangeFromPoint(x, y)
        return r && { node: r.startContainer, offset: r.startOffset }
    }
    return null
}

/**
 * Where the reader is: the block on the top line of the scroll box, and how far into its text
 * that line starts
 * @param {Element} container the scroll box
 * @param {Element} article the element the volume's markup is in
 * @param {Element[]} blocks getBlocks(article)
 * @returns {{block: number, char: number}|null} null when no block is on screen, so the last
 *  position found is kept rather than one made up
 */
export function anchorAtTop(container, article, blocks) {
    const top = container.getBoundingClientRect().top + 2
    const index = blocks.findIndex(b => b.getBoundingClientRect().bottom > top)
    if (index < 0) return null
    const block = blocks[index]
    if (block.getBoundingClientRect().top >= top) return { block: index, char: 0 }

    // the block starts above the screen: find the line it's on at the top
    const articleRect = article.getBoundingClientRect()
    const x = articleRect.left + parseFloat(getComputedStyle(article).paddingLeft || 0) + 2
    const caret = caretAt(article.ownerDocument, x, top)
    if (!caret) return { block: index, char: 0 }
    const el = caret.node.nodeType === Node.TEXT_NODE ? caret.node.parentElement : caret.node
    // the caret can be in a block inside this one, a paragraph in a list item
    const inner = el?.closest(BLOCK_SELECTOR)
    if (!inner || !block.contains(inner)) return { block: index, char: 0 }
    const innerIndex = blocks.indexOf(inner)
    return { block: innerIndex, char: textOffsetIn(inner, caret.node, caret.offset) }
}

/**
 * The top of an anchor's line, in client coordinates
 * @returns {number|null} null when the block isn't there
 */
function anchorTop(blocks, anchor) {
    const block = blocks[anchor.block]
    if (!block) return null
    const range = anchor.char > 0 ? rangeAt(block, anchor.char) : null
    const rect = range?.getClientRects()[0] ?? block.getBoundingClientRect()
    return rect.top
}

/**
 * Scroll so an anchor's line is at the top of the scroll box
 * @param {Element} container the scroll box
 * @param {Element[]} blocks getBlocks of the volume
 * @param {{block: number, char: number}} anchor where to go
 * @param {number} margin pixels to leave above the line
 * @returns {boolean} false when the anchor points past the volume's blocks
 */
export function scrollToAnchor(container, blocks, anchor, margin = 0) {
    const top = anchorTop(blocks, anchor)
    if (top === null) return false
    container.scrollTop += top - container.getBoundingClientRect().top - margin
    return true
}

/**
 * Which block a tap puts the bookmark above: the tapped block when the tap is in its top half,
 * the next one when it's in the bottom half
 * @param {number} blockTop the tapped block's top
 * @param {number} blockHeight the tapped block's height
 * @param {number} tapY where the tap was
 * @param {number} index the tapped block's index
 * @param {number} count how many blocks the volume has
 * @returns {number}
 */
export function bookmarkBlockForTap(blockTop, blockHeight, tapY, index, count) {
    if (tapY - blockTop < blockHeight / 2) return index
    return Math.min(index + 1, count - 1)
}

/**
 * The block a tap between blocks, in the gap, puts the bookmark above: the first one below it
 * @param {Element[]} blocks getBlocks of the volume
 * @param {number} tapY where the tap was
 * @returns {number} -1 when the tap is below every block
 */
export function blockBelow(blocks, tapY) {
    return blocks.findIndex(b => b.getBoundingClientRect().top >= tapY)
}
