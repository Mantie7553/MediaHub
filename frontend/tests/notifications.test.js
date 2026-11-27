import { describe, expect, it } from "vitest"
import { badgeCount, notificationKind, sortNotifications, visibleNotifications } from "../src/utils/notifications"

const items = [
    { id: "old-read", created_at: "2026-11-01T10:00:00Z", read_at: "2026-11-02T10:00:00Z" },
    { id: "new-read", created_at: "2026-11-20T10:00:00Z", read_at: "2026-11-21T10:00:00Z" },
    { id: "old-unread", created_at: "2026-11-05T10:00:00Z", read_at: null },
    { id: "new-unread", created_at: "2026-11-25T10:00:00Z", read_at: null },
]

describe("notification order", () => {
    it("lists unread first, newest first in each", () => {
        expect(sortNotifications(items).map(n => n.id)).toEqual(["new-unread", "old-unread", "new-read", "old-read"])
    })

    it("doesn't change the list it was given", () => {
        const copy = [...items]
        sortNotifications(items)
        expect(items).toEqual(copy)
    })
})

describe("notification views", () => {
    it("hides the read ones in the unread view", () => {
        expect(visibleNotifications(items, "unread").map(n => n.id)).toEqual(["new-unread", "old-unread"])
    })

    it("shows every one in the all view", () => {
        expect(visibleNotifications(items, "all")).toHaveLength(4)
    })
})

describe("notification kinds", () => {
    it("knows each kind the server sends", () => {
        for (const kind of ["approval_needed", "download_failed", "downloading", "download_by_hand"]) {
            expect(notificationKind(kind).label).not.toBe("Notification")
        }
    })

    it("falls back for a kind it doesn't know", () => {
        expect(notificationKind("something_new").label).toBe("Notification")
    })
})

describe("badge count", () => {
    it("caps at 99+", () => {
        expect(badgeCount(5)).toBe("5")
        expect(badgeCount(100)).toBe("99+")
    })
})
