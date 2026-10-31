// @vitest-environment jsdom
import { describe, expect, it } from "vitest"
import {
    blockBelow, bookmarkBlockForTap, getBlocks, rangeAt, scrollToAnchor, textOffsetIn,
} from "../src/utils/readerAnchor"

function page(html) {
    const article = document.createElement("article")
    article.innerHTML = html
    return article
}

// gives an element a fixed box, jsdom lays nothing out
function boxAt(el, top, height) {
    el.getBoundingClientRect = () => ({ top, bottom: top + height, height, left: 0, right: 0, width: 0 })
}

describe("finding blocks", () => {
    it("lists paragraphs, headings, images and list items in reading order", () => {
        const article = page('<h1>One</h1><div><p>a</p><img src="x"></div><ul><li><p>b</p></li></ul>')
        expect(getBlocks(article).map(b => b.tagName)).toEqual(["H1", "P", "IMG", "LI", "P"])
    })
})

describe("text offsets", () => {
    it("counts characters across inline markup", () => {
        const article = page("<p>She closed <i>the door</i>. He waited.</p>")
        const p = article.querySelector("p")
        const italic = p.querySelector("i").firstChild
        expect(textOffsetIn(p, italic, 4)).toBe("She closed the ".length)
    })

    it("finds the same character it counted", () => {
        const article = page("<p>She closed <i>the door</i>. He waited.</p>")
        const p = article.querySelector("p")
        const char = "She closed the door. ".length
        const range = rangeAt(p, char)
        expect(range.toString()).toBe("H")
        expect(textOffsetIn(p, range.startContainer, range.startOffset)).toBe(char)
    })

    it("takes the last character for an offset past the end", () => {
        const p = page("<p>Short.</p>").querySelector("p")
        expect(rangeAt(p, 99).toString()).toBe(".")
    })

    it("has no range in a block without text", () => {
        expect(rangeAt(page('<img src="x">').querySelector("img"), 0)).toBeNull()
    })
})

describe("scrolling to an anchor", () => {
    it("puts the block's top at the top of the scroll box, less the margin", () => {
        const container = document.createElement("div")
        const article = page("<p>a</p><p>b</p>")
        container.appendChild(article)
        const blocks = getBlocks(article)
        boxAt(container, 50, 500)
        boxAt(blocks[1], 850, 40)
        container.scrollTop = 0
        expect(scrollToAnchor(container, blocks, { block: 1, char: 0 }, 24)).toBe(true)
        expect(container.scrollTop).toBe(850 - 50 - 24)
    })

    it("refuses an anchor past the volume", () => {
        const container = document.createElement("div")
        expect(scrollToAnchor(container, [], { block: 3, char: 0 })).toBe(false)
    })
})

describe("placing a bookmark", () => {
    it("goes above the tapped paragraph from its top half", () => {
        expect(bookmarkBlockForTap(100, 80, 130, 4, 10)).toBe(4)
    })

    it("goes above the next paragraph from the bottom half", () => {
        expect(bookmarkBlockForTap(100, 80, 150, 4, 10)).toBe(5)
    })

    it("stays above the last paragraph at the end of the volume", () => {
        expect(bookmarkBlockForTap(100, 80, 170, 9, 10)).toBe(9)
    })

    it("goes above the paragraph below a tap in the gap", () => {
        const blocks = getBlocks(page("<p>a</p><p>b</p><p>c</p>"))
        blocks.forEach((b, i) => boxAt(b, i * 100, 80))
        expect(blockBelow(blocks, 90)).toBe(1)
        expect(blockBelow(blocks, 290)).toBe(-1)
    })
})
