import { afterEach, describe, expect, it, vi } from "vitest"
import { readStored, writeStored } from "../src/utils/storage"

function memoryStorage() {
    const data = new Map()
    return {
        getItem: key => data.has(key) ? data.get(key) : null,
        setItem: (key, value) => data.set(key, String(value)),
    }
}

describe("stored preferences", () => {
    afterEach(() => {
        vi.unstubAllGlobals()
    })

    it("reads back what was written", () => {
        vi.stubGlobal("localStorage", memoryStorage())
        writeStored("tab", "manga")
        expect(readStored("tab", "anime")).toBe("manga")
    })

    it("falls back when nothing is stored", () => {
        vi.stubGlobal("localStorage", memoryStorage())
        expect(readStored("tab", "anime")).toBe("anime")
    })

    it("falls back when the stored value is not allowed", () => {
        vi.stubGlobal("localStorage", memoryStorage())
        writeStored("tab", "podcasts")
        expect(readStored("tab", "anime", ["anime", "manga"])).toBe("anime")
    })

    it("falls back instead of throwing when storage is blocked", () => {
        vi.stubGlobal("localStorage", {
            getItem: () => { throw new Error("SecurityError") },
            setItem: () => { throw new Error("SecurityError") },
        })
        expect(() => writeStored("tab", "manga")).not.toThrow()
        expect(readStored("tab", "anime")).toBe("anime")
    })
})
