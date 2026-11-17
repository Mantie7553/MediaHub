import { describe, expect, it } from "vitest"
import { albumFromTrack, artistFromTrack, artistFromUploader, canonical, suggestions } from "../src/utils/suggest"

const artists = ["Ado", "Aimer", "LiSA", "YOASOBI", "Kenshi Yonezu"]

describe("suggestions", () => {
    it("puts names that start with the text before ones that contain it", () => {
        expect(suggestions(artists, "a")).toEqual(["Ado", "Aimer", "LiSA", "YOASOBI"])
    })

    it("ignores case", () => {
        expect(suggestions(artists, "yo")).toEqual(["YOASOBI", "Kenshi Yonezu"])
    })

    it("offers everything, up to the max, before anything is typed", () => {
        expect(suggestions(artists, "", 2)).toEqual(["Ado", "Aimer"])
    })

    it("doesn't suggest what is already typed", () => {
        expect(suggestions(artists, "lisa")).toEqual([])
    })
})

describe("canonical names", () => {
    it("uses the library's spelling when only case or spacing differs", () => {
        expect(canonical(artists, " yoasobi ")).toBe("YOASOBI")
        expect(canonical(artists, "kenshi   yonezu")).toBe("Kenshi Yonezu")
    })

    it("keeps a new name as typed", () => {
        expect(canonical(artists, " Eve ")).toBe("Eve")
    })
})

describe("artist from a YouTube channel", () => {
    it("drops the - Topic suffix", () => {
        expect(artistFromUploader("Aimer - Topic")).toBe("Aimer")
    })

    it("keeps other names", () => {
        expect(artistFromUploader("YOASOBI and Echoes")).toBe("YOASOBI and Echoes")
        expect(artistFromUploader(undefined)).toBe("")
    })
})

describe("artist from a search result", () => {
    it("takes the first credited artist", () => {
        expect(artistFromTrack({ artists: ["Kendrick Lamar", "Taylour Paige"], uploader: "Kendrick Lamar" })).toBe("Kendrick Lamar")
    })

    it("falls back to the channel", () => {
        expect(artistFromTrack({ artists: [], uploader: "Aimer - Topic" })).toBe("Aimer")
        expect(artistFromTrack({ uploader: "Final Move" })).toBe("Final Move")
    })
})

describe("album from a search result", () => {
    it("keeps an album track's album", () => {
        expect(albumFromTrack({ title: "We Cry Together", album: "Mr. Morale & The Big Steppers" })).toBe("Mr. Morale & The Big Steppers")
    })

    it("leaves a single empty so it goes under Singles", () => {
        expect(albumFromTrack({ title: "Not Like Us", album: "Not Like Us" })).toBe("")
        expect(albumFromTrack({ title: "Not like us ", album: "Not Like Us" })).toBe("")
    })

    it("is empty for videos", () => {
        expect(albumFromTrack({ title: "Not Like Us" })).toBe("")
    })
})
