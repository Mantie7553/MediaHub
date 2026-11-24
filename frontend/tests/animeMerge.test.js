import { describe, expect, it } from "vitest"
import { matchSeries, parseSeason, seasonLabel } from "../src/utils/animeMerge"

describe("what a merged title stood for", () => {
    it("names seasons, specials and whole series", () => {
        expect(seasonLabel({ season: 2 })).toBe("season 2")
        expect(seasonLabel({ season: 0 })).toBe("specials")
        expect(seasonLabel({ season: -1 })).toBe("series")
        // a Sonarr title is the series whatever entry it was linked to
        expect(seasonLabel({ season: 1, sonarr: true })).toBe("series")
    })
})

describe("attaching a title to a series", () => {
    it("reads the season typed", () => {
        expect(parseSeason("1")).toBe(1)
        expect(parseSeason(" 0 ")).toBe(0)
        expect(parseSeason("")).toBeNull()
        expect(parseSeason("-1")).toBeNull()
        expect(parseSeason("1.5")).toBeNull()
        expect(parseSeason("abc")).toBeNull()
    })

    it("lists the series matching what was typed, never the title itself", () => {
        const series = [
            { id: "a", title: "Gintama", tvdb_id: 79089 },
            { id: "b", title: "Monogatari", tvdb_id: 102261 },
            { id: "self", title: "Gintama (2006)", tvdb_id: 1 },
        ]
        expect(matchSeries(series, "gin", "self").map(s => s.id)).toEqual(["a"])
        expect(matchSeries(series, "", "self").map(s => s.id)).toEqual(["a", "b"])
        expect(matchSeries(null, "x", "self")).toEqual([])
    })
})
