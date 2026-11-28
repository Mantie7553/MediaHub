import { isHomePath } from "../../utils/homePages"
import { BarChart3, Bell, Compass, Download, LayoutDashboard, Settings, Users } from "lucide-react"

/**
 * The app's main pages, shared by the desktop sidebar and the mobile tab bar
 * @param {string} role the user's role, admins get the admin pages too
 * @returns { path, title, short, icon, group, isActive?, badge?, mobile? } where group is "main", "admin"
 *  or "footer". isActive(pathname), when given, decides when the item is highlighted instead of the
 *  path. badge "notifications" shows the unread count beside it. mobile false keeps it off the
 *  phone tab bar (the bell sits in the top bar there)
 */
export default function navItems(role) {
    return [
        // "/" opens the user's home page, so Home is lit on every home page
        { path: "/", title: "Home", short: "Home", icon: LayoutDashboard, group: "main", isActive: isHomePath },
        { path: "/discover", title: "Discover", short: "Discover", icon: Compass, group: "main" },
        { path: "/notifications", title: "Notifications", short: "Alerts", icon: Bell, group: "main", badge: "notifications", mobile: false },
        ...(role === "admin" ? [
            { path: "/downloads", title: "Downloads", short: "Downloads", icon: Download, group: "admin" },
            { path: "/users", title: "Users", short: "Users", icon: Users, group: "admin" },
            { path: "/insights", title: "Insights", short: "Insights", icon: BarChart3, group: "admin" },
        ] : []),
        { path: "/settings", title: "Settings", short: "Settings", icon: Settings, group: "footer" },
    ]
}
