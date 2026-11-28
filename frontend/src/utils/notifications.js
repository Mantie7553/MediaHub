import { AlertTriangle, BookOpen, CheckCircle2, Download, Inbox, PlayCircle, Sparkles, XCircle } from "lucide-react"

/**
 * How each kind of notification looks: its icon, a short label and the tone of its icon
 */
export const notificationKinds = {
    approval_needed: { icon: Inbox, label: "Needs approval", tone: "text-warning bg-warning/12" },
    download_failed: { icon: AlertTriangle, label: "Failed", tone: "text-error bg-error/12" },
    downloading: { icon: Download, label: "Downloading", tone: "text-info bg-info/12" },
    download_by_hand: { icon: BookOpen, label: "Download by hand", tone: "text-secondary bg-secondary/12" },
    request_approved: { icon: CheckCircle2, label: "Approved", tone: "text-success bg-success/12" },
    request_rejected: { icon: XCircle, label: "Not approved", tone: "text-error bg-error/12" },
    title_ready: { icon: PlayCircle, label: "Ready", tone: "text-primary bg-primary/12" },
    new_parts: { icon: Sparkles, label: "New", tone: "text-accent bg-accent/12" },
}

/**
 * The notification settings, in the order they're listed, each with what it turns off. Admin
 * kinds only show for admins, the server only sends the kinds a user can get
 */
export const notificationSettings = [
    { kind: "title_ready", title: "Your requests are ready", hint: "When something you asked for arrives." },
    { kind: "new_parts", title: "New episodes and chapters", hint: "For titles you requested, wishlisted or are watching or reading." },
    { kind: "request_approved", title: "Requests approved", hint: "When an admin approves something you asked for." },
    { kind: "request_rejected", title: "Requests not approved", hint: "When an admin turns down a request, and why." },
    { kind: "approval_needed", title: "Requests to approve", hint: "Someone asked for something that needs your approval." },
    { kind: "download_by_hand", title: "Downloads by hand", hint: "Light novels to fetch and add yourself." },
    { kind: "downloading", title: "Downloads started", hint: "New content started downloading." },
    { kind: "download_failed", title: "Failed downloads", hint: "A download stopped with an error." },
]

/**
 * The settings a user has, in order: the ones the server sent them
 * @param {Record<string, boolean>} prefs kind → on, from /me/notification-prefs
 */
export function settingsFor(prefs) {
    return notificationSettings.filter(s => s.kind in (prefs ?? {}))
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
