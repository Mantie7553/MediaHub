import { AlertTriangle, BookOpen, Download, Inbox } from "lucide-react"

/**
 * How each kind of notification looks: its icon, a short label and the tone of its icon
 */
export const notificationKinds = {
    approval_needed: { icon: Inbox, label: "Needs approval", tone: "text-warning bg-warning/12" },
    download_failed: { icon: AlertTriangle, label: "Failed", tone: "text-error bg-error/12" },
    downloading: { icon: Download, label: "Downloading", tone: "text-info bg-info/12" },
    download_by_hand: { icon: BookOpen, label: "Download by hand", tone: "text-secondary bg-secondary/12" },
}

/**
 * How a notification looks, a plain inbox for a kind this version doesn't know
 * @param {string} kind the notification's kind
 */
export function notificationKind(kind) {
    return notificationKinds[kind] ?? { icon: Inbox, label: "Notification", tone: "text-base-content/70 bg-white/5" }
}

/**
 * Notifications in the order they're listed: unread first, newest first in each
 * @param {object[]} items notifications with created_at and read_at
 */
export function sortNotifications(items) {
    return [...items].sort((a, b) =>
        (!b.read_at) - (!a.read_at) || new Date(b.created_at) - new Date(a.created_at))
}

/**
 * The notifications a view shows: only the unread ones, or all of them
 * @param {object[]} items notifications
 * @param {string} view "unread" or "all"
 */
export function visibleNotifications(items, view) {
    const sorted = sortNotifications(items)
    return view === "unread" ? sorted.filter(n => !n.read_at) : sorted
}

/**
 * The bell's count: up to 99, then "99+"
 * @param {number} n unread notifications
 */
export function badgeCount(n) {
    return n > 99 ? "99+" : String(n)
}
