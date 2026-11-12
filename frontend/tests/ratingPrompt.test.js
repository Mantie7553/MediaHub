import { describe, expect, it } from "vitest"
import { RECENT_COMPLETION_MS, dismissKey, ratingWord, shouldPromptRating } from "../src/utils/ratingPrompt"

const now = Date.parse("2026-11-11T12:00:00Z")
const entry = (fields = {}) => ({ id: "e1", status: "completed", rating: null, updated_at: "2026-11-11T11:59:00Z", ...fields })

describe("asking for a rating", () => {
    it("asks when a title is completed while the page is open", () => {
        expect(shouldPromptRating("current", entry(), now, false)).toBe(true)
        expect(shouldPromptRating("planned", entry(), now, false)).toBe(true)
        // ticked the last part of a title that wasn't on the list
        expect(shouldPromptRating(null, entry(), now, false)).toBe(true)
    })

    it("doesn't ask when it was completed already", () => {
        expect(shouldPromptRating("completed", entry(), now, false)).toBe(false)
    })

    it("doesn't ask for rated, unfinished or missing entries", () => {
        expect(shouldPromptRating("current", entry({ rating: 7 }), now, false)).toBe(false)
        expect(shouldPromptRating("planned", entry({ status: "current" }), now, false)).toBe(false)
        expect(shouldPromptRating("current", undefined, now, false)).toBe(false)
    })

    it("doesn't ask again after Not now", () => {
        expect(shouldPromptRating("current", entry(), now, true)).toBe(false)
        expect(shouldPromptRating(undefined, entry(), now, true)).toBe(false)
    })

    it("asks on opening the page only about recent completions", () => {
        expect(shouldPromptRating(undefined, entry(), now, false)).toBe(true)
        const old = new Date(now - RECENT_COMPLETION_MS - 1000).toISOString()
        expect(shouldPromptRating(undefined, entry({ updated_at: old }), now, false)).toBe(false)
        expect(shouldPromptRating(undefined, entry({ updated_at: null }), now, false)).toBe(false)
    })

    it("remembers Not now per completion", () => {
        expect(dismissKey(entry())).not.toBe(dismissKey(entry({ updated_at: "2026-11-12T00:00:00Z" })))
    })

    it("has a word for every rating", () => {
        for (let r = 1; r <= 10; r++) expect(ratingWord(r)).not.toBe("")
        expect(ratingWord(null)).toBe("")
        expect(ratingWord(0)).toBe("")
    })
})
