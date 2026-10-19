import { describe, expect, it } from "vitest"
import Format from "../src/utils/format"

describe("short date times", () => {
    it("leaves the year off dates in this year", () => {
        const date = new Date(new Date().getFullYear(), 9, 18, 21, 38)
        const text = Format.shortDateTime(date)
        expect(text).toContain("Oct 18")
        expect(text).toContain("9:38")
        expect(text).not.toContain(String(date.getFullYear()))
    })

    it("shows the year on dates from other years", () => {
        const text = Format.shortDateTime(new Date(2020, 0, 5, 8, 5))
        expect(text).toContain("Jan 5")
        expect(text).toContain("2020")
    })
})
