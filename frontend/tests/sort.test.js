import { describe, expect, it } from "vitest"
import { byTitle, compareValues, date, number, pickOption, sortItems, text, thenBy } from "../src/utils/sort"
import * as sortOptions from "../src/utils/sortOptions"

const titles = list => list.map(item => item.title)

describe("comparing values", () => {
    it("puts missing values last whichever way the sort runs", () => {
        const items = [{ r: 5 }, { r: null }, { r: 9 }, {}, { r: 1 }]
        expect(sortItems(items, { compare: number(i => i.r) }).map(i => i.r)).toEqual([1, 5, 9, null, undefined])
        expect(sortItems(items, { compare: number(i => i.r, true) }).map(i => i.r)).toEqual([9, 5, 1, null, undefined])
    })

    it("sorts numbers inside names as numbers, ignoring case", () => {
        const items = [{ title: "vol 10" }, { title: "Vol 2" }, { title: "Vol 1" }]
        expect(titles(sortItems(items, { compare: text(i => i.title) }))).toEqual(["Vol 1", "Vol 2", "vol 10"])
    })

    it("sorts dates newest first, with no date last", () => {
        const items = [
            { title: "a", d: "2020-01-01" }, { title: "b", d: null },
            { title: "c", d: "2024-05-01T00:00:00Z" }, { title: "d", d: "not a date" },
        ]
        expect(titles(sortItems(items, { compare: date(i => i.d, true) }))).toEqual(["c", "a", "b", "d"])
    })

    it("breaks ties with the next comparator", () => {
        const items = [
            { title: "B", artist: "Y" }, { title: "A", artist: "Y" }, { title: "C", artist: "X" },
        ]
        const compare = thenBy(text(i => i.artist), byTitle)
        expect(titles(sortItems(items, { compare }))).toEqual(["C", "A", "B"])
    })

    it("treats equal values as equal", () => {
        expect(compareValues(3, 3)).toBe(0)
        expect(compareValues(null, undefined)).toBe(0)
    })
})

describe("sorting a list", () => {
    it("leaves the list it was given alone", () => {
        const items = [{ title: "b" }, { title: "a" }]
        const sorted = sortItems(items, { compare: byTitle })
        expect(titles(sorted)).toEqual(["a", "b"])
        expect(titles(items)).toEqual(["b", "a"])
    })

    it("keeps the given order for an option without a comparator", () => {
        const items = [{ title: "b" }, { title: "a" }]
        expect(sortItems(items, { key: "custom" })).toBe(items)
    })

    it("falls back to the first option for a key it doesn't offer", () => {
        const options = [{ key: "title" }, { key: "rating-high" }]
        expect(pickOption(options, "rating-high").key).toBe("rating-high")
        expect(pickOption(options, "gone").key).toBe("title")
        expect(pickOption(options, "").key).toBe("title")
    })
})

describe("sort options", () => {
    const entry = (title, extra) => ({ media_type: "anime", status: "current", title, media_title: title, ...extra })

    it("ranks the user's rating high to low, unrated last", () => {
        const items = [entry("A", { rating: 6 }), entry("B", { rating: null }), entry("C", { rating: 9 })]
        const option = pickOption(sortOptions.trackedSorts("anime"), "rating-high")
        expect(titles(sortItems(items, option))).toEqual(["C", "A", "B"])
        const low = pickOption(sortOptions.trackedSorts("anime"), "rating-low")
        expect(titles(sortItems(items, low))).toEqual(["A", "C", "B"])
    })

    it("ranks progress by how far through, with an unknown length after known ones", () => {
        const items = [
            entry("Half", { progress_count: 6, progress_total: 12, progress_total_known: true }),
            entry("Unknown", { progress_count: 30, progress_total: 30, progress_total_known: false }),
            entry("Most", { progress_count: 11, progress_total: 12, progress_total_known: true }),
        ]
        const option = pickOption(sortOptions.trackedSorts("anime"), "progress-most")
        expect(titles(sortItems(items, option))).toEqual(["Most", "Half", "Unknown"])
    })

    it("looks up the user's entry for a library item", () => {
        const items = [{ id: "1", title: "One" }, { id: "2", title: "Two" }]
        const userContentMap = { 2: { rating: 8 }, 1: { rating: 3 } }
        const option = pickOption(sortOptions.librarySorts("anime"), "rating-high")
        expect(titles(sortItems(items, option, { userContentMap }))).toEqual(["Two", "One"])
    })

    it("offers no progress sorts for movies", () => {
        expect(sortOptions.trackedSorts("movie").some(o => o.key.startsWith("progress"))).toBe(false)
    })

    it("sorts a playlist by artist, then album, then track number", () => {
        const items = [
            { title: "s3", artist: "B", album_title: "X", track_number: 1 },
            { title: "s2", artist: "A", album_title: "Z", track_number: 1 },
            { title: "s1", artist: "A", album_title: "Y", track_number: 2 },
            { title: "s0", artist: "A", album_title: "Y", track_number: 1 },
        ]
        const option = pickOption(sortOptions.playlistSorts, "artist-album")
        expect(titles(sortItems(items, option))).toEqual(["s0", "s1", "s2", "s3"])
        const byAlbum = pickOption(sortOptions.playlistSorts, "album-artist")
        expect(titles(sortItems(items, byAlbum))).toEqual(["s3", "s0", "s1", "s2"])
    })

    it("puts parts not done yet first, in number order", () => {
        const items = [{ title: "1", n: 1 }, { title: "2", n: 2 }, { title: "3", n: 3 }]
        const ctx = { numberOf: i => i.n, isDone: i => i.n === 2 || i.n === 1 }
        const options = sortOptions.partSorts("Unread")
        expect(titles(sortItems(items, pickOption(options, "undone"), ctx))).toEqual(["3", "1", "2"])
        expect(titles(sortItems(items, pickOption(options, "newest"), ctx))).toEqual(["3", "2", "1"])
        expect(pickOption(options, "undone").label).toBe("Unread first")
    })

    it("gives every option set unique keys and comparators that run on sparse items", () => {
        const sets = Object.entries(sortOptions).flatMap(([name, value]) => {
            if (Array.isArray(value)) return [[name, value]]
            if (name === "partSorts") return [[name, value("Unread")]]
            if (typeof value === "function") return [[name, value("anime")], [`${name}(movie)`, value("movie")]]
            return []
        })
        const ctx = { userContentMap: {}, numberOf: () => null, isDone: () => false }
        const sparse = [{ id: "a" }, { id: "b", title: "x", rating: 3 }, { id: "c", status: "completed", media_type: "movie" }]
        for (const [name, options] of sets) {
            const keys = options.map(o => o.key)
            expect(new Set(keys).size, name).toBe(keys.length)
            for (const option of options) {
                expect(() => sortItems(sparse, option, ctx), `${name} ${option.key}`).not.toThrow()
            }
        }
    })
})
