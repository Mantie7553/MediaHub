import { describe, expect, it } from "vitest"
import { ALWAYS_SHOWN_ABOVE, filterByQuery, hideOnScroll, matchesQuery } from "../src/utils/search"

const items = [
    { id: "1", type: "anime", title: "Frieren" },
    { id: "2", type: "movie", title: "Your Name" },
    { id: "3", type: "manga", title: "Pokémon Adventures" },
    { id: "4", title: "The Book", artist: "YOASOBI" },
]

describe("matching a search", () => {
    it("ignores case and accents", () => {
        expect(matchesQuery(items[2], "POKEMON")).toBe(true)
    })

    it("needs every word, in any order, from the title or artist", () => {
        expect(matchesQuery(items[3], "yoasobi book")).toBe(true)
        expect(matchesQuery(items[3], "yoasobi idol")).toBe(false)
    })

    it("matches everything before anything is typed", () => {
        expect(matchesQuery(items[0], "  ")).toBe(true)
    })

    it("uses a list entry's title", () => {
        expect(matchesQuery({ media_title: "Frieren" }, "fri")).toBe(true)
    })
})

describe("filtering by a search", () => {
    it("keeps the matches in their order", () => {
        expect(filterByQuery(items, "en").map(item => item.id)).toEqual(["1", "3"])
    })
})

describe("hiding the search bar on scroll", () => {
    const deep = ALWAYS_SHOWN_ABOVE + 500

    it("hides when scrolling down", () => {
        expect(hideOnScroll(false, deep, deep + 40)).toEqual({ hidden: true, moved: true })
    })

    it("comes back when scrolling up", () => {
        expect(hideOnScroll(true, deep, deep - 40)).toEqual({ hidden: false, moved: true })
    })

    it("ignores small movements", () => {
        expect(hideOnScroll(true, deep, deep - 3)).toEqual({ hidden: true, moved: false })
    })

    it("always shows near the top of the page", () => {
        expect(hideOnScroll(true, ALWAYS_SHOWN_ABOVE + 50, 20)).toEqual({ hidden: false, moved: true })
        expect(hideOnScroll(false, 0, 60)).toEqual({ hidden: false, moved: true })
    })
})
