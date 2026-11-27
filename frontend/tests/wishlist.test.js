import { describe, expect, it } from "vitest"
import { wishlistKeys, wishlistMessage, wishlistedId } from "../src/utils/wishlist"

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

describe("telling the user what wishlisting did", () => {
    it("says a download started or waits for an admin", () => {
        expect(wishlistMessage({ download: "requested" })).toEqual({ type: "success", message: "Added to your wishlist, download started" })
        expect(wishlistMessage({ download: "pending" }).message).toMatch(/waiting for an admin/)
    })

    it("warns with the reason when the last download failed", () => {
        const msg = wishlistMessage({ download: "failed", reason: "MangaDex hosts no English chapters" })
        expect(msg.type).toBe("warning")
        expect(msg.message).toContain("MangaDex hosts no English chapters")
    })

    it("warns when an admin turned the request down", () => {
        expect(wishlistMessage({ download: "rejected" }).type).toBe("warning")
    })

    it("falls back to a plain message for an answer it doesn't know", () => {
        expect(wishlistMessage({})).toEqual({ type: "success", message: "Added to your wishlist" })
        expect(wishlistMessage(undefined)).toEqual({ type: "success", message: "Added to your wishlist" })
    })
})
