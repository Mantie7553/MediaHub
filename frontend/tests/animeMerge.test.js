import { describe, expect, it } from "vitest"
import { seasonLabel } from "../src/utils/animeMerge"

describe("what a merged title stood for", () => {
    it("names seasons, specials and whole series", () => {
        expect(seasonLabel({ season: 2 })).toBe("season 2")
        expect(seasonLabel({ season: 0 })).toBe("specials")
        expect(seasonLabel({ season: -1 })).toBe("series")
        // a Sonarr title is the series whatever entry it was linked to
        expect(seasonLabel({ season: 1, sonarr: true })).toBe("series")
    })
})
