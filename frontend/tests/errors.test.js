import { describe, expect, it } from "vitest"
import { errorMessage, isNotFound } from "../src/utils/errors"

// the shapes axios produces: request without response when the server can't be reached
const noResponse = { request: {}, message: "Network Error" }
const withStatus = (status, data) => ({ request: {}, response: { status, data }, message: `Request failed with status code ${status}` })

describe("errorMessage", () => {
    it("explains an unreachable server", () => {
        expect(errorMessage(noResponse)).toMatch(/can't reach the server/i)
    })

    it.each([502, 503, 504])("explains a %i from the proxy", (status) => {
        expect(errorMessage(withStatus(status, "Bad Gateway"))).toMatch(/restarting or unreachable/)
    })

    it("says not found for a 404", () => {
        expect(errorMessage(withStatus(404, "not found"))).toBe("Not found")
    })

    it("uses the backend's plain text message, capitalized and trimmed", () => {
        expect(errorMessage(withStatus(400, "invalid invite code\n"))).toBe("Invalid invite code")
    })

    it("uses an error field from a JSON body", () => {
        expect(errorMessage(withStatus(409, { error: "already requested" }))).toBe("Already requested")
    })

    it("falls back when the body says nothing useful", () => {
        expect(errorMessage(withStatus(500, ""), "Could not save")).toBe("Could not save")
        expect(errorMessage(withStatus(500, new Blob()), "Could not load the page")).toBe("Could not load the page")
    })

    it("passes plain strings through", () => {
        expect(errorMessage("Playback session expired")).toBe("Playback session expired")
    })

    it("has a default for nothing at all", () => {
        expect(errorMessage(null)).toBe("Something went wrong")
        expect(errorMessage(new Error("boom"))).toBe("Something went wrong")
    })
})

describe("isNotFound", () => {
    it("is true only for a 404", () => {
        expect(isNotFound(withStatus(404, ""))).toBe(true)
        expect(isNotFound(withStatus(500, ""))).toBe(false)
        expect(isNotFound(noResponse)).toBe(false)
        expect(isNotFound(null)).toBe(false)
    })
})
