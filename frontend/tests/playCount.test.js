import { describe, expect, it } from "vitest"
import { trackListening, PLAY_AFTER_SECS } from "../src/utils/playCount"

// feeds positions one by one, returning the final state and how many plays were counted
function listen(positions, { trackId = "t1", duration = 200, state = null } = {}) {
    let plays = 0
    for (const time of positions) {
        const result = trackListening(state, trackId, time, duration)
        state = result.state
        if (result.counted) plays++
    }
    return { state, plays }
}

const steadily = (from, to, step = 0.25) => Array.from({ length: Math.round((to - from) / step) + 1 }, (_, i) => from + i * step)

describe("counting plays", () => {
    it("counts a play once enough has been heard, and only once", () => {
        expect(listen(steadily(0, PLAY_AFTER_SECS - 1)).plays).toBe(0)
        expect(listen(steadily(0, 120)).plays).toBe(1)
    })

    it("needs only half of a short track", () => {
        expect(listen(steadily(0, 11), { duration: 20 }).plays).toBe(1)
        expect(listen(steadily(0, 9), { duration: 20 }).plays).toBe(0)
    })

    it("doesn't count seeking ahead as listening", () => {
        expect(listen([0, 0.25, 0.5, 60, 120, 180]).plays).toBe(0)
    })

    it("starts over for a different track", () => {
        const first = listen(steadily(0, 20))
        expect(listen(steadily(0, 20), { trackId: "t2", state: first.state }).plays).toBe(0)
    })

    it("counts a repeat as a new play", () => {
        const once = listen(steadily(0, 199), { duration: 200 })
        expect(once.plays).toBe(1)
        expect(listen(steadily(0, 40), { duration: 200, state: once.state }).plays).toBe(1)
    })
})
