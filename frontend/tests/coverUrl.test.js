import { describe, expect, it } from "vitest"
import { coverUrl } from "../src/utils/coverUrl"

const proxied = url => `/api/proxy/cover?url=${encodeURIComponent(url)}`

describe("cover URLs", () => {
    it("loads TMDB posters at a smaller size than the original", () => {
        expect(coverUrl("https://image.tmdb.org/t/p/original/abc.jpg"))
            .toBe("https://image.tmdb.org/t/p/w780/abc.jpg")
    })

    it("leaves TMDB posters already at a size alone", () => {
        expect(coverUrl("https://image.tmdb.org/t/p/w500/abc.jpg")).toBe("https://image.tmdb.org/t/p/w500/abc.jpg")
    })

    it("proxies MangaDex covers at their 512px size", () => {
        expect(coverUrl("https://uploads.mangadex.org/covers/m1/c.png", "mangadex"))
            .toBe(proxied("https://uploads.mangadex.org/covers/m1/c.png.512.jpg"))
    })

    it("keeps a MangaDex size that is already chosen", () => {
        expect(coverUrl("https://uploads.mangadex.org/covers/m1/c.png.256.jpg", "mangadex"))
            .toBe(proxied("https://uploads.mangadex.org/covers/m1/c.png.256.jpg"))
    })

    it("leaves our own and other covers alone", () => {
        expect(coverUrl("/api/media/1/cover?media_token=x", "mangadex")).toBe("/api/media/1/cover?media_token=x")
        expect(coverUrl("https://s4.anilist.co/file/cover.jpg", "anilist")).toBe("https://s4.anilist.co/file/cover.jpg")
        expect(coverUrl(null)).toBe(null)
    })
})
