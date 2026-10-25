import { describe, expect, it } from "vitest"
import { watchedBetween } from "../src/utils/watchTime"

// feeds positions one by one, returning the seconds counted as watched
function watch(positions) {
    let last = null
    let watched = 0
    for (const time of positions) {
        watched += watchedBetween(last, time)
        last = time
    }
    return watched
}

const steadily = (from, to, step = 0.25) => Array.from({ length: Math.round((to - from) / step) + 1 }, (_, i) => from + i * step)

describe("watchedBetween", () => {
    it("counts playing time", () => {
        expect(watch(steadily(0, 60))).toBeCloseTo(60)
    })

    it("doesn't count a seek ahead", () => {
        expect(watch([...steadily(0, 10), ...steadily(600, 610)])).toBeCloseTo(20)
    })

    it("doesn't count going back", () => {
        expect(watch([...steadily(100, 110), ...steadily(5, 15)])).toBeCloseTo(20)
    })

    it("counts faster playback", () => {
        // timeupdate every 250ms at 2x
        expect(watch(steadily(0, 30, 0.5))).toBeCloseTo(30)
    })

    it("ignores the first update and bad positions", () => {
        expect(watchedBetween(null, 50)).toBe(0)
        expect(watchedBetween(10, NaN)).toBe(0)
        expect(watchedBetween(NaN, 10)).toBe(0)
    })
})
