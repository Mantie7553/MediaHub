import { describe, expect, it } from "vitest"
import { filterRecords, statusCounts } from "../src/utils/records"

const items = [
    { id: 1, status: "failed", title: "Frieren" },
    { id: 2, status: "complete", title: "Dandadan" },
    { id: 3, status: "failed", title: "Blue Lock" },
    { id: 4, status: "weird", title: "Mystery" },
]
const titleOf = item => item.title

describe("status counts", () => {
    it("counts each status in the given order and skips empty ones", () => {
        expect(statusCounts(items, ["queued", "complete", "failed"])).toEqual([
            { key: "complete", count: 1 },
            { key: "failed", count: 2 },
            { key: "weird", count: 1 },
        ])
    })

    it("is empty with no records", () => {
        expect(statusCounts([], ["failed"])).toEqual([])
    })
})

describe("filtering records", () => {
    it("keeps one status", () => {
        expect(filterRecords(items, "failed", "", titleOf).map(i => i.id)).toEqual([1, 3])
    })

    it("matches titles ignoring case and spaces around the query", () => {
        expect(filterRecords(items, "all", "  blue ", titleOf).map(i => i.id)).toEqual([3])
    })

    it("combines status and title", () => {
        expect(filterRecords(items, "complete", "frieren", titleOf)).toEqual([])
    })

    it("treats a missing title as empty", () => {
        expect(filterRecords([{ status: "failed" }], "all", "x", () => null)).toEqual([])
    })
})
