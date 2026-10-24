import { BarChart3, Compass, Download, LayoutDashboard, Settings, Users } from "lucide-react"

/**
 * The app's main pages, shared by the desktop sidebar and the mobile tab bar
 * @param {string} role the user's role, admins get the admin pages too
 * @returns { path, title, short, icon, group } where group is "main", "admin" or "footer"
 */
export default function navItems(role) {
    return [
        { path: "/", title: "Dashboard", short: "Home", icon: LayoutDashboard, group: "main" },
        { path: "/discover", title: "Discover", short: "Discover", icon: Compass, group: "main" },
        ...(role === "admin" ? [
            { path: "/downloads", title: "Downloads", short: "Downloads", icon: Download, group: "admin" },
            { path: "/users", title: "Users", short: "Users", icon: Users, group: "admin" },
            { path: "/insights", title: "Insights", short: "Insights", icon: BarChart3, group: "admin" },
        ] : []),
        { path: "/settings", title: "Settings", short: "Settings", icon: Settings, group: "footer" },
    ]
}
