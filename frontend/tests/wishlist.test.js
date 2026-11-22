import { describe, expect, it } from "vitest"
import { wishlistKeys, wishlistedId } from "../src/utils/wishlist"

const keys = wishlistKeys([
    { media_item_id: "m1", external_source: "anilist", external_id: "123" },
    { media_item_id: "m2", external_source: null, external_id: null },
])

describe("finding a title on the wishlist", () => {
    it("matches library titles and list entries by media id", () => {
        expect(wishlistedId(keys, { id: "m1" })).toBe("m1")
        expect(wishlistedId(keys, { media_item_id: "m2" })).toBe("m2")
    })

    it("matches search results by their source's id", () => {
        expect(wishlistedId(keys, { external_source: "anilist", external_id: 123 })).toBe("m1")
    })

    it("doesn't mix up the same id from another source", () => {
        expect(wishlistedId(keys, { external_source: "tmdb", external_id: 123 })).toBeUndefined()
    })

    it("finds nothing for titles not on it", () => {
        expect(wishlistedId(keys, { id: "m3" })).toBeUndefined()
        expect(wishlistedId(keys, undefined)).toBeUndefined()
    })
})
