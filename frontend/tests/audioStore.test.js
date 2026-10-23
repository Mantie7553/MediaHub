import { beforeEach, describe, expect, it } from "vitest"
import useAudioStore from "../src/stores/useAudioStore"

const tracks = ["a", "b", "c", "d", "e"].map(id => ({ id, title: id }))
const current = () => useAudioStore.getState().currentTrack?.id

describe("audio queue", () => {
    beforeEach(() => {
        useAudioStore.getState().stop()
        useAudioStore.setState({ shuffle: false, repeat: "off" })
    })

    it("plays through the queue in order", () => {
        useAudioStore.getState().playAlbum(tracks, 1)
        useAudioStore.getState().next()
        expect(current()).toBe("c")
        useAudioStore.getState().prev()
        useAudioStore.getState().prev()
        expect(current()).toBe("a")
    })

    it("stays on the first track at the start", () => {
        useAudioStore.getState().playAlbum(tracks, 0)
        useAudioStore.getState().prev()
        expect(current()).toBe("a")
    })

    it("pauses on the first track after the last one", () => {
        useAudioStore.getState().playAlbum(tracks, 4)
        useAudioStore.getState().next()
        expect(current()).toBe("a")
        expect(useAudioStore.getState().isPlaying).toBe(false)
    })

    it("keeps playing from the start with repeat all, and goes back to the last track", () => {
        useAudioStore.setState({ repeat: "all" })
        useAudioStore.getState().playAlbum(tracks, 4)
        useAudioStore.getState().next()
        expect(current()).toBe("a")
        expect(useAudioStore.getState().isPlaying).toBe(true)
        useAudioStore.getState().prev()
        expect(current()).toBe("e")
    })

    it("cycles repeat off, all, one", () => {
        const { cycleRepeat } = useAudioStore.getState()
        cycleRepeat()
        expect(useAudioStore.getState().repeat).toBe("all")
        cycleRepeat()
        expect(useAudioStore.getState().repeat).toBe("one")
        cycleRepeat()
        expect(useAudioStore.getState().repeat).toBe("off")
    })

    it("shuffles the rest of the queue after the current track, playing each once", () => {
        useAudioStore.getState().playAlbum(tracks, 2)
        useAudioStore.getState().toggleShuffle()
        expect(current()).toBe("c")

        const played = [current()]
        for (let i = 1; i < tracks.length; i++) {
            useAudioStore.getState().next()
            played.push(current())
        }
        expect(played[0]).toBe("c")
        expect([...played].sort()).toEqual(["a", "b", "c", "d", "e"])
    })

    it("starts a new queue shuffled when shuffle is on, from the track picked", () => {
        useAudioStore.setState({ shuffle: true })
        useAudioStore.getState().playAlbum(tracks, 3)
        expect(current()).toBe("d")
        expect(useAudioStore.getState().order[0]).toBe(3)
    })

    it("carries on in order from the current track when shuffle is turned off", () => {
        useAudioStore.setState({ shuffle: true })
        useAudioStore.getState().playAlbum(tracks, 0)
        useAudioStore.getState().next()
        const at = tracks.findIndex(t => t.id === current())
        useAudioStore.getState().toggleShuffle()
        expect(current()).toBe(tracks[at].id)
        useAudioStore.getState().next()
        // the track after it in the album, or the first one again after the last
        expect(current()).toBe(tracks[(at + 1) % tracks.length].id)
    })

    it("plays a single track as a queue of one", () => {
        useAudioStore.getState().play(tracks[0])
        useAudioStore.getState().next()
        expect(current()).toBe("a")
        expect(useAudioStore.getState().isPlaying).toBe(false)
    })
})
