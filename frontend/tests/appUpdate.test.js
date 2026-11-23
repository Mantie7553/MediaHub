import { afterEach, describe, expect, it, vi } from "vitest"
import { fetchBuildId, isNewBuild } from "../src/utils/appUpdate"

describe("isNewBuild", () => {
    it("is new only when the server's build differs", () => {
        expect(isNewBuild("abc", "abc")).toBe(false)
        expect(isNewBuild("abc", "def")).toBe(true)
    })

    it("never counts an unknown build as new", () => {
        expect(isNewBuild("abc", null)).toBe(false)
        expect(isNewBuild(null, "def")).toBe(false)
    })
})

describe("fetchBuildId", () => {
    afterEach(() => vi.unstubAllGlobals())

    it("reads the build from version.json", async () => {
        vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true, json: async () => ({ build: "abc" }) }))
        expect(await fetchBuildId()).toBe("abc")
    })

    it("gives null when the file is missing", async () => {
        vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: false, json: async () => ({}) }))
        expect(await fetchBuildId()).toBeNull()
    })

    it("gives null when offline", async () => {
        vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("offline")))
        expect(await fetchBuildId()).toBeNull()
    })
})
