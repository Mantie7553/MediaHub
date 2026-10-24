import { describe, expect, it } from "vitest"
import { niceMax, tooltipAlign } from "../src/components/charts/chartScale"

describe("chart scale tops", () => {
    it("rounds up to 1, 2 or 5 times a power of ten", () => {
        expect(niceMax(0)).toBe(1)
        expect(niceMax(1)).toBe(1)
        expect(niceMax(3)).toBe(5)
        expect(niceMax(7)).toBe(10)
        expect(niceMax(12)).toBe(20)
        expect(niceMax(480)).toBe(500)
    })

    it("keeps tooltips inside the chart near its edges", () => {
        expect(tooltipAlign(0, 30)).toBe("-translate-x-3")
        expect(tooltipAlign(15, 30)).toBe("-translate-x-1/2")
        expect(tooltipAlign(29, 30)).toContain("100%")
    })
})
