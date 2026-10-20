// @vitest-environment jsdom
import { describe, expect, it } from "vitest"
import { sanitizeHtml } from "../src/utils/sanitize"

describe("sanitizing outside HTML", () => {
    it("keeps formatting and images", () => {
        const html = '<p>Chapter <i>one</i><br><img src="/api/light-novels/1/volumes/2/images/a.jpg?media_token=x" alt="art"></p>'
        expect(sanitizeHtml(html)).toBe(html)
    })

    it("removes scripts and event handlers", () => {
        const out = sanitizeHtml('<img src="x" onerror="steal()"><script>steal()</script><a href="javascript:steal()">x</a>')
        expect(out).not.toMatch(/steal|onerror|script/)
    })

    it("removes stylesheets, forms and frames", () => {
        const out = sanitizeHtml('<style>body{display:none}</style><form action="/x"><input name="p"></form><iframe src="https://evil"></iframe><p>text</p>')
        expect(out).toBe("<p>text</p>")
    })

    it("treats missing content as empty", () => {
        expect(sanitizeHtml(null)).toBe("")
    })
})
