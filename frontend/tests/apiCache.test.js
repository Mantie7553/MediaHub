import { beforeEach, describe, expect, it } from "vitest"
import { clearCached, forgetCached, getCached, setCached } from "../src/services/apiCache"

describe("api cache", () => {
    beforeEach(() => clearCached())

    it("returns what a path last loaded", () => {
        setCached("/media/1", { title: "old" })
        setCached("/media/1", { title: "new" })
        expect(getCached("/media/1")).toEqual({ title: "new" })
        expect(getCached("/media/2")).toBeUndefined()
    })

    it("keeps empty answers, which still count as loaded", () => {
        setCached("/me/media", [])
        expect(getCached("/me/media")).toEqual([])
    })

    it("forgets one path or everything", () => {
        setCached("/a", 1)
        setCached("/b", 2)
        forgetCached("/a")
        expect(getCached("/a")).toBeUndefined()
        expect(getCached("/b")).toBe(2)
        clearCached()
        expect(getCached("/b")).toBeUndefined()
    })

    it("drops the least recently loaded path past 100", () => {
        for (let i = 0; i <= 100; i++) setCached(`/p${i}`, i)
        expect(getCached("/p0")).toBeUndefined()
        expect(getCached("/p1")).toBe(1)
        expect(getCached("/p100")).toBe(100)

        // loading a path again makes it the newest
        setCached("/p1", 1)
        setCached("/new", 0)
        expect(getCached("/p1")).toBe(1)
        expect(getCached("/p2")).toBeUndefined()
    })
})
