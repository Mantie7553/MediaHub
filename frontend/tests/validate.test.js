import { describe, expect, it } from "vitest"
import { validatePassword } from "../src/utils/validate"

describe("password rules", () => {
    it("accepts a password that meets every rule", () => {
        expect(validatePassword("Correct-Horse-9")).toBeNull()
    })

    it("names the first rule that fails", () => {
        expect(validatePassword("Short-1")).toMatch(/at least 12/)
        expect(validatePassword("no-upper-case-1")).toMatch(/uppercase/)
        expect(validatePassword("No-Number-Here")).toMatch(/number/)
    })

    it("refuses passwords longer than the server can hash", () => {
        expect(validatePassword("Aa1!" + "x".repeat(68))).toBeNull()
        expect(validatePassword("Aa1!" + "x".repeat(69))).toMatch(/72/)
    })
})
