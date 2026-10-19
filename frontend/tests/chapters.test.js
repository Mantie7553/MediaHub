import { describe, expect, it } from "vitest"
import { chapterNumberFromName, isChapterArchive } from "../src/utils/chapters"

// the same cases as backend/internal/mangafiles, so the preview matches what the server saves
describe("chapter numbers from file names", () => {
    const cases = [
        ["[0098] Chapter 98 - Katana Man.cbz", "", "98"],
        ["[0052.5] Chapter 52.5.cbz", "", "52.5"],
        ["Chainsaw Man c098.cbz", "Chainsaw Man", "98"],
        ["Chainsaw Man v12 c098 (Digital) [Group].cbz", "Chainsaw Man", "98"],
        ["Chainsaw_Man_Ch.98.zip", "Chainsaw Man", "98"],
        ["Chapter 101.5.cbz", "", "101.5"],
        ["ch 7.cbz", "", "7"],
        ["#12.cbz", "", "12"],
        ["Chainsaw Man 098.cbz", "Chainsaw Man", "98"],
        ["Kaiju No. 8 012.cbz", "Kaiju No. 8", "12"],
        ["Chainsaw Man v12.cbz", "Chainsaw Man", null],
        ["Chainsaw Man Vol. 3 (Digital).cbz", "Chainsaw Man", null],
        ["cover.cbz", "", null],
    ]
    for (const [name, series, want] of cases) {
        it(`${name} -> ${want}`, () => {
            expect(chapterNumberFromName(name, series)).toBe(want)
        })
    }
})

describe("chapter archives", () => {
    it("accepts cbz and zip only", () => {
        expect(isChapterArchive("a.CBZ")).toBe(true)
        expect(isChapterArchive("a.zip")).toBe(true)
        expect(isChapterArchive("a.cbr")).toBe(false)
        expect(isChapterArchive("a.pdf")).toBe(false)
    })
})
