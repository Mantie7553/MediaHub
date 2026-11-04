import { afterEach, describe, expect, it, vi } from "vitest"
import { subtitleMode } from "../src/utils/subtitles"

const ass = { url: "/subs_3.vtt", assUrl: "/subs_3.ass", burn: false }
const srt = { url: "/subs_4.vtt", assUrl: null, burn: false }
const pgs = { url: null, assUrl: null, burn: true }

describe("subtitleMode", () => {
    // node has Worker-less globals, so stand in for a browser that can run the renderer
    function withRenderer() {
        vi.stubGlobal("Worker", class {})
        vi.stubGlobal("OffscreenCanvas", class {})
    }
    afterEach(() => vi.unstubAllGlobals())

    it("is off without a track", () => {
        expect(subtitleMode(null)).toBe("off")
    })

    it("burns image tracks in", () => {
        withRenderer()
        expect(subtitleMode(pgs, { videoFullscreen: true })).toBe("burn")
    })

    it("renders ASS once its file is here", () => {
        withRenderer()
        expect(subtitleMode(ass, { assReady: true })).toBe("ass")
        expect(subtitleMode(ass, { assReady: false })).toBe("vtt")
    })

    it("falls back to WebVTT when ASS can't be drawn", () => {
        withRenderer()
        expect(subtitleMode(ass, { assReady: true, assFailed: true })).toBe("vtt")
        expect(subtitleMode(ass, { assReady: true, videoFullscreen: true })).toBe("vtt")
    })

    it("falls back to WebVTT in a browser without the renderer's needs", () => {
        vi.stubGlobal("OffscreenCanvas", undefined)
        expect(subtitleMode(ass, { assReady: true })).toBe("vtt")
    })

    it("shows other text tracks as WebVTT", () => {
        withRenderer()
        expect(subtitleMode(srt, { assReady: true })).toBe("vtt")
    })
})
