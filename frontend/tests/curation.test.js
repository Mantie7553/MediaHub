import { describe, expect, it } from "vitest"
import {
    changedFields, dateInputValue, descriptionToText, episodeFromName, genresFromText,
    textToDescription, trackFromName, volumeFromName,
} from "../src/utils/curation"

describe("episode numbers from file names", () => {
    const cases = [
        ["Frieren - S01E05 - Phantoms of the Dead.mkv", "1", "5"],
        ["frieren.s02e112.1080p.mkv", "2", "112"],
        ["Show 1x05.mp4", "1", "5"],
        ["[SubsPlease] Frieren - 05 (1080p) [ABCD1234].mkv", "", "5"],
        ["[Group] Show - 12v2 [720p].mkv", "", "12"],
        ["Episode 7.mp4", "", "7"],
        ["Movie (2020).mkv", "", ""],
    ]
    it.each(cases)("%s", (name, season, episode) => {
        expect(episodeFromName(name)).toEqual({ season, episode })
    })
})

describe("volume numbers from file names", () => {
    it.each([
        ["Mushoku Tensei Volume 3.epub", "3"],
        ["mushoku_tensei_volume_12.epub", "12"],
        ["Spice and Wolf v07.epub", "7"],
        ["Spice and Wolf Vol. 2.epub", "2"],
        ["Side Story.epub", ""],
    ])("%s", (name, number) => expect(volumeFromName(name)).toBe(number))
})

describe("tracks from file names", () => {
    it.each([
        ["03 - Song Name.mp3", "3", "Song Name"],
        ["03. Song Name.mp3", "3", "Song Name"],
        ["12 Song_Name.mp3", "12", "Song Name"],
        ["Song Name.mp3", "", "Song Name"],
        ["1999.mp3", "", "1999"],
    ])("%s", (name, trackNumber, title) => expect(trackFromName(name)).toEqual({ trackNumber, title }))
})

describe("descriptions", () => {
    it("shows HTML line breaks as new lines and saves them back", () => {
        const saved = "First line<br>Second <i>line</i><br/>Third"
        const text = descriptionToText(saved, true)
        expect(text).toBe("First line\nSecond <i>line</i>\nThird")
        expect(textToDescription(text, true)).toBe("First line<br>Second <i>line</i><br>Third")
    })
    it("leaves plain text alone", () => {
        expect(descriptionToText("a\nb", false)).toBe("a\nb")
        expect(textToDescription(" a\nb ", false)).toBe("a\nb")
    })
    it("clears an empty description", () => {
        expect(textToDescription("   ", true)).toBeNull()
        expect(descriptionToText(null, true)).toBe("")
    })
})

describe("genres", () => {
    it("splits, trims and drops repeats", () => {
        expect(genresFromText(" Action, fantasy,, Fantasy ,Drama")).toEqual(["Action", "fantasy", "Drama"])
        expect(genresFromText("")).toEqual([])
    })
})

describe("changed fields", () => {
    it("sends only what changed, clearing empty text and converting numbers", () => {
        const initial = { title: "A", studio: "Madhouse", runtime_mins: "90", genres: ["X"] }
        const values = { title: "A ", studio: "", runtime_mins: "95", genres: ["X"] }
        expect(changedFields(initial, values, ["runtime_mins"])).toEqual({ title: "A", studio: null, runtime_mins: 95 })
    })
    it("sends nothing when nothing changed", () => {
        expect(changedFields({ title: "A" }, { title: "A" })).toEqual({})
    })
})

describe("date inputs", () => {
    it.each([
        ["2023-09-29T00:00:00Z", "2023-09-29"],
        ["0001-01-01T00:00:00Z", ""],
        [null, ""],
    ])("%s", (value, want) => expect(dateInputValue(value)).toBe(want))
})
