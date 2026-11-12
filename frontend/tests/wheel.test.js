import { describe, expect, it } from "vitest"
import { clampValue, itemLook, scrollForValue, valueAtScroll, wheelLabel } from "../src/utils/wheel"

describe("wheel picker maths", () => {
    it("keeps values in range and whole", () => {
        expect(clampValue(-2, 0, 10)).toBe(0)
        expect(clampValue(14, 0, 10)).toBe(10)
        expect(clampValue(6.6, 0, 10)).toBe(7)
        expect(clampValue(undefined, 0, 10)).toBe(0)
    })

    it("turns scroll positions into the value in the middle and back", () => {
        expect(valueAtScroll(0, 44, 0, 10)).toBe(0)
        expect(valueAtScroll(44 * 7, 44, 0, 10)).toBe(7)
        // past half a slot rounds to the nearer one
        expect(valueAtScroll(44 * 3 + 20, 44, 0, 10)).toBe(3)
        expect(valueAtScroll(44 * 3 + 24, 44, 0, 10)).toBe(4)
        // overscroll stays in range
        expect(valueAtScroll(44 * 40, 44, 0, 10)).toBe(10)
        expect(scrollForValue(7, 44, 0, 10)).toBe(308)
        expect(scrollForValue(5, 44, 1, 10)).toBe(176)
        expect(valueAtScroll(scrollForValue(5, 44, 1, 10), 44, 1, 10)).toBe(5)
    })

    it("labels the unset value", () => {
        expect(wheelLabel(0, 0, "–")).toBe("–")
        expect(wheelLabel(8, 0, "–")).toBe("8")
        expect(wheelLabel(0, null, "–")).toBe("0")
    })

    it("shrinks and fades numbers away from the middle, to a floor", () => {
        expect(itemLook(0)).toEqual({ scale: 1, opacity: 1 })
        expect(itemLook(-1).scale).toBeLessThan(1)
        expect(itemLook(1)).toEqual(itemLook(-1))
        expect(itemLook(10)).toEqual(itemLook(3))
        expect(itemLook(3).opacity).toBeGreaterThan(0)
    })
})
