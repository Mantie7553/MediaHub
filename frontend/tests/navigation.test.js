import { describe, expect, it, vi } from "vitest"
import { exitViewer, openViewer, switchViewer } from "../src/utils/navigation"

describe("viewer navigation", () => {
    it("opening a viewer marks it as coming from the details page", () => {
        const navigate = vi.fn()
        openViewer(navigate, "/watch/movie/1", { position: 30 })
        expect(navigate).toHaveBeenCalledWith("/watch/movie/1", { state: { position: 30, fromDetails: true } })
    })

    it("moving to the next chapter replaces the entry and keeps the state", () => {
        const navigate = vi.fn()
        const location = { state: { fromDetails: true, animeId: "a", position: 120 } }
        switchViewer(navigate, location, "/watch/episode/2", { position: 0 })
        expect(navigate).toHaveBeenCalledWith("/watch/episode/2", {
            replace: true,
            state: { fromDetails: true, animeId: "a", position: 0 },
        })
    })

    it("leaving a viewer opened from details goes back to that entry", () => {
        const navigate = vi.fn()
        exitViewer(navigate, { state: { fromDetails: true } }, "/manga/1")
        expect(navigate).toHaveBeenCalledWith(-1)
    })

    it("leaving a viewer opened directly replaces it with the details page", () => {
        const navigate = vi.fn()
        exitViewer(navigate, { state: null }, "/manga/1")
        expect(navigate).toHaveBeenCalledWith("/manga/1", { replace: true })
    })

    it("without a details page it goes home when there's no history", () => {
        const navigate = vi.fn()
        vi.stubGlobal("window", { history: { state: { idx: 0 } } })
        exitViewer(navigate, { state: null }, null)
        expect(navigate).toHaveBeenCalledWith("/")
        vi.unstubAllGlobals()
    })
})
