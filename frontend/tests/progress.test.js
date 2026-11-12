import { describe, expect, it } from "vitest"
import { OPEN_PROGRESS_MAX, hasParts, partsLabel, progressMeta, progressPercent, progressRange } from "../src/utils/progress"

describe("progress counts", () => {
    it("knows which types count parts", () => {
        expect(hasParts("anime")).toBe(true)
        expect(hasParts("manga")).toBe(true)
        expect(hasParts("light_novel")).toBe(true)
        expect(hasParts("movie")).toBe(false)
        expect(hasParts("music_album")).toBe(false)
        expect(partsLabel("light_novel")).toBe("Volumes read")
    })

    it("goes up to the source's total for a list entry", () => {
        // 20 chapters downloaded of 100
        expect(progressRange({ progress_total: 100, progress_total_known: true })).toEqual({ max: 100, known: true })
    })

    it("stops at the files when the source doesn't say how long it is", () => {
        expect(progressRange({ progress_total: 20, progress_total_known: false })).toEqual({ max: 20, known: false })
        // a count given before stays reachable
        expect(progressRange({ progress_count: 30, progress_total: 20, progress_total_known: false }).max).toBe(30)
        // nothing downloaded and no known length: nothing to pick
        expect(progressRange({ progress_count: 0, progress_total: 0, progress_total_known: false }).max).toBe(0)
    })

    it("uses a search result's total, or stays open without one", () => {
        expect(progressRange(null, { total_parts: 24 })).toEqual({ max: 24, known: true })
        expect(progressRange(null, { title: "x" })).toEqual({ max: OPEN_PROGRESS_MAX, known: false })
    })

    it("labels cards with the count and the total when it's known", () => {
        expect(progressMeta("manga", { progress_count: 50, progress_total: 100, progress_total_known: true })).toBe("Ch 50 / 100")
        expect(progressMeta("light_novel", { progress_count: 3, progress_total: 20, progress_total_known: false })).toBe("Vol 3")
        expect(progressMeta("manga", { progress_count: 0, progress_total: 100, progress_total_known: true })).toBeNull()
        expect(progressMeta("movie", { progress_count: 1 })).toBeNull()
    })

    it("fills the bar only against a known total", () => {
        expect(progressPercent({ progress_count: 50, progress_total: 100, progress_total_known: true })).toBe(50)
        expect(progressPercent({ progress_count: 50, progress_total: 60, progress_total_known: false })).toBe(0)
        expect(progressPercent({ progress_count: 120, progress_total: 100, progress_total_known: true })).toBe(100)
    })
})
