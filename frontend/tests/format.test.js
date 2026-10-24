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

describe("byte sizes", () => {
    it("picks the unit and keeps one decimal only for small values", () => {
        expect(Format.bytes(0)).toBe("0 B")
        expect(Format.bytes(1023)).toBe("1023 B")
        expect(Format.bytes(1536)).toBe("1.5 KB")
        expect(Format.bytes(820 * 1024 ** 2)).toBe("820 MB")
        expect(Format.bytes(1.4 * 1024 ** 4)).toBe("1.4 TB")
    })

    it("shows a dash when the size is unknown", () => {
        expect(Format.bytes(null)).toBe("—")
        expect(Format.bytes(undefined)).toBe("—")
    })
})

describe("relative times", () => {
    const now = new Date(2026, 9, 23, 12, 0)
    const ago = secs => new Date(now - secs * 1000)

    it("counts minutes, hours and days", () => {
        expect(Format.relativeTime(ago(20), now)).toBe("just now")
        expect(Format.relativeTime(ago(5 * 60), now)).toBe("5 min ago")
        expect(Format.relativeTime(ago(3 * 3600), now)).toBe("3 h ago")
        expect(Format.relativeTime(ago(30 * 3600), now)).toBe("yesterday")
        expect(Format.relativeTime(ago(4 * 86400), now)).toBe("4 days ago")
    })

    it("falls back to the date after a month, and Never without one", () => {
        expect(Format.relativeTime(new Date(2026, 6, 1), now)).toContain("July 1")
        expect(Format.relativeTime(null, now)).toBe("Never")
    })
})

describe("calendar days", () => {
    it("reads the API's day as a local date, not the day before", () => {
        expect(Format.day("2026-10-23")).toBe("Oct 23")
        expect(Format.day("2026-01-01")).toBe("Jan 1")
    })
})
