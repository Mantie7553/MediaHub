import { describe, expect, it } from "vitest"
import { bufferedEnd, cueHtml, pictureRect, seekFraction, segmentMarks, subtitleLift } from "../src/utils/player"

function ranges(...pairs) {
    return { length: pairs.length, start: i => pairs[i][0], end: i => pairs[i][1] }
}

describe("segment marks", () => {
    it("places segments as percentages", () => {
        expect(segmentMarks([{ type: "op", start: 60, end: 150 }], 1200)).toEqual([{ type: "op", left: 5, width: 7.5 }])
    })
    it("clamps to the video and drops empty ones", () => {
        expect(segmentMarks([
            { type: "ed", start: 1100, end: 1300 },
            { type: "recap", start: 50, end: 50 },
            { type: "preview", start: 1250, end: 1300 },
        ], 1200)).toEqual([{ type: "ed", left: (1100 / 1200) * 100, width: (100 / 1200) * 100 }])
    })
    it("has none before the length is known", () => {
        expect(segmentMarks([{ type: "op", start: 0, end: 90 }], 0)).toEqual([])
        expect(segmentMarks([{ type: "op", start: 0, end: 90 }], NaN)).toEqual([])
        expect(segmentMarks([{ type: "op", start: 0, end: 90 }], Infinity)).toEqual([])
    })
})

describe("buffered end", () => {
    it("finds the range playing", () => {
        expect(bufferedEnd(ranges([0, 30], [100, 160]), 120)).toBe(160)
    })
    it("counts a range starting just ahead", () => {
        expect(bufferedEnd(ranges([100.3, 160]), 100)).toBe(160)
    })
    it("is 0 when nothing is loaded there", () => {
        expect(bufferedEnd(ranges([0, 30]), 60)).toBe(0)
        expect(bufferedEnd(undefined, 10)).toBe(0)
    })
})

describe("seek fraction", () => {
    const rect = { left: 100, width: 400 }
    it("is how far along the bar", () => {
        expect(seekFraction(200, rect)).toBe(0.25)
    })
    it("clamps to the bar", () => {
        expect(seekFraction(0, rect)).toBe(0)
        expect(seekFraction(900, rect)).toBe(1)
    })
    it("is 0 for a bar with no width", () => {
        expect(seekFraction(50, { left: 0, width: 0 })).toBe(0)
    })
})

describe("picture rect", () => {
    it("letterboxes a wide video in a tall box", () => {
        expect(pictureRect({ width: 400, height: 800 }, 1920, 1080)).toEqual({ left: 0, top: 287.5, width: 400, height: 225 })
    })
    it("pillarboxes a narrow video in a wide box", () => {
        expect(pictureRect({ width: 1000, height: 300 }, 400, 300)).toEqual({ left: 300, top: 0, width: 400, height: 300 })
    })
    it("is the whole box before the video's size is known", () => {
        expect(pictureRect({ width: 800, height: 450 }, 0, 0)).toEqual({ left: 0, top: 0, width: 800, height: 450 })
    })
})

describe("subtitle lift", () => {
    it("is the controls' height when the picture fills the box", () => {
        expect(subtitleLift(96, { width: 1920, height: 1080 }, 1920, 1080)).toBe(96)
    })
    it("is nothing when the band under the picture fits the controls", () => {
        expect(subtitleLift(96, { width: 400, height: 800 }, 1920, 1080)).toBe(0)
    })
    it("is the part of the controls over the picture", () => {
        expect(subtitleLift(96, { width: 1920, height: 1160 }, 1920, 1080)).toBe(56)
    })
})

describe("cue html", () => {
    it("keeps italics, bold and underline", () => {
        expect(cueHtml("<i>Hello</i> <b.loud>there</b> <u>you</u>")).toBe("<i>Hello</i> <b>there</b> <u>you</u>")
    })
    it("drops other tags", () => {
        expect(cueHtml("<v Roger>Hi <c.yellow>there</c><00:01.000>!")).toBe("Hi there!")
        expect(cueHtml("<script>alert(1)</script><img src=x onerror=alert(1)>")).toBe("alert(1)")
    })
    it("keeps entities and escapes the rest", () => {
        expect(cueHtml("Tom &amp; Jerry & friends")).toBe("Tom &amp; Jerry &amp; friends")
        expect(cueHtml("3 < 4")).toBe("3 &lt; 4")
        expect(cueHtml("5 > 2")).toBe("5 &gt; 2")
    })
    it("breaks lines", () => {
        expect(cueHtml("one\ntwo")).toBe("one<br>two")
    })
})
