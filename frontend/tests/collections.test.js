import { describe, expect, it } from "vitest"
import { addBody, applyMove, canHold, dropTarget, itemKind, moveTarget, shuffled } from "../src/utils/collections"

const entries = ["a", "b", "c", "d"].map(id => ({ entry_id: id }))
const order = list => list.map(e => e.entry_id).join("")

describe("what goes where", () => {
    it("lets playlists take tracks and albums only", () => {
        expect(canHold("playlist", "music_track")).toBe(true)
        expect(canHold("playlist", "album")).toBe(true)
        expect(canHold("playlist", "anime")).toBe(false)
        expect(canHold("playlist", "playlist")).toBe(false)
    })

    it("lets collections take titles, albums, tracks and playlists, but not collections", () => {
        for (const kind of ["anime", "movie", "manga", "light_novel", "music_track", "album", "playlist"]) {
            expect(canHold("collection", kind)).toBe(true)
        }
        expect(canHold("collection", "collection")).toBe(false)
    })

    it("names items the way the rules do", () => {
        expect(itemKind({ media_type: "music_album" })).toBe("album")
        expect(itemKind({ type: "light_novel" })).toBe("light_novel")
        expect(itemKind({ kind: "playlist", type: "whatever" })).toBe("playlist")
    })

    it("sends the right id for each kind", () => {
        expect(addBody({ type: "album", id: "al" })).toEqual({ album_id: "al" })
        expect(addBody({ media_type: "music_album", album_id: "al", id: "entry" })).toEqual({ album_id: "al" })
        expect(addBody({ kind: "playlist", id: "pl" })).toEqual({ collection_id: "pl" })
        expect(addBody({ media_type: "anime", media_item_id: "m", id: "entry" })).toEqual({ media_item_id: "m" })
        expect(addBody({ type: "music_track", id: "t" })).toEqual({ media_item_id: "t" })
    })
})

describe("moving an entry a step", () => {
    it("goes below the one two up when moving up, or to the top from second", () => {
        expect(moveTarget(entries, 2, "up")).toBe("a")
        expect(moveTarget(entries, 1, "up")).toBe(null)
        expect(moveTarget(entries, 0, "up")).toBe(undefined)
    })

    it("goes below the next one when moving down", () => {
        expect(moveTarget(entries, 1, "down")).toBe("c")
        expect(moveTarget(entries, 3, "down")).toBe(undefined)
    })

    it("moves to the top from anywhere but the top", () => {
        expect(moveTarget(entries, 3, "top")).toBe(null)
        expect(moveTarget(entries, 0, "top")).toBe(undefined)
    })

    it("gives the order the server will have", () => {
        expect(order(applyMove(entries, 2, moveTarget(entries, 2, "up")))).toBe("acbd")
        expect(order(applyMove(entries, 1, moveTarget(entries, 1, "down")))).toBe("acbd")
        expect(order(applyMove(entries, 3, null))).toBe("dabc")
    })
})

describe("dropping a dragged entry", () => {
    it("lands where it was dropped, moving either way", () => {
        expect(order(applyMove(entries, 0, dropTarget(entries, 0, 2)))).toBe("bcad")
        expect(order(applyMove(entries, 3, dropTarget(entries, 3, 1)))).toBe("adbc")
        expect(order(applyMove(entries, 2, dropTarget(entries, 2, 0)))).toBe("cabd")
    })

    it("does nothing when dropped where it was", () => {
        expect(dropTarget(entries, 1, 1)).toBe(undefined)
    })
})

describe("shuffling", () => {
    it("keeps every item and leaves the original alone", () => {
        const list = [1, 2, 3, 4, 5]
        const out = shuffled(list, () => 0.5)
        expect([...out].sort()).toEqual(list)
        expect(list).toEqual([1, 2, 3, 4, 5])
    })
})
