import { describe, expect, it } from "vitest"
import { homePage, homePageForDiscover, homePages, homePath, isHomePath } from "../src/utils/homePages"
import navItems from "../src/components/layout/navItems"

describe("home pages", () => {
    it("each stored home_page opens its own page", () => {
        expect(homePath("overview")).toBe("/home")
        expect(homePath("anime")).toBe("/anime")
        expect(homePath("movie")).toBe("/movies")
        expect(homePath("music")).toBe("/music")
        expect(homePath("light_novel")).toBe("/light-novels")
    })

    it("an unknown or missing setting falls back to the overview", () => {
        expect(homePath("podcasts")).toBe("/home")
        expect(homePath(undefined)).toBe("/home")
        expect(homePage(null).key).toBe("overview")
    })

    it("matches the values the server accepts", () => {
        expect(homePages.map(page => page.key)).toEqual(["overview", "anime", "movie", "manga", "music", "light_novel"])
    })

    it("finds the home page for a Discover tab", () => {
        expect(homePageForDiscover("music").path).toBe("/music")
        expect(homePageForDiscover("light_novel").path).toBe("/light-novels")
        expect(homePageForDiscover("nope")).toBeUndefined()
    })
})

describe("Home in the navigation", () => {
    const home = navItems("user").find(item => item.path === "/")

    it("is lit on every home page", () => {
        for (const path of ["/", "/home", "/anime", "/movies", "/manga", "/music", "/light-novels", "/music/"]) {
            expect(home.isActive(path)).toBe(true)
            expect(isHomePath(path)).toBe(true)
        }
    })

    it("is not lit on details pages or other pages", () => {
        for (const path of ["/anime/12", "/manga/3/chapters/1/read", "/discover", "/settings", "/homepage"]) {
            expect(home.isActive(path)).toBe(false)
        }
    })
})
