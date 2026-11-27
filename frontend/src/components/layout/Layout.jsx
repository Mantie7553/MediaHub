import { useEffect } from "react";
import { NavLink } from "react-router-dom";
import { Bell } from "lucide-react";
import Sidebar, { Avatar } from "./Sidebar";
import Logo from "./Logo";
import MobileNav from "./MobileNav";
import useMe from "../../hooks/useMe";
import { getRole } from "../../utils/auth";
import useNotificationStore from "../../stores/useNotificationStore";
import { badgeCount } from "../../utils/notifications";

// how often the bell's count is checked
const NOTIFICATION_POLL_MS = 30000;

/**
 * Component to layout our pages properly. The Sidebar is always on the left on desktop, phones
 * get a tab bar along the bottom and the sidebar as a drawer behind their avatar
 * @param {any} children the other page contents
 * @returns
 */
export default function Layout({children}) {
    const { me } = useMe();
    // only admins get notifications for now
    const isAdmin = (me?.role ?? getRole()) === "admin";
    const unread = useNotificationStore(state => state.unread);
    const refreshUnread = useNotificationStore(state => state.refresh);

    useEffect(() => {
        if (!isAdmin) return;
        refreshUnread();
        const interval = setInterval(refreshUnread, NOTIFICATION_POLL_MS);
        return () => clearInterval(interval);
    }, [isAdmin, refreshUnread]);

    return <div className="drawer lg:drawer-open">
        <input id="my-drawer-3" type="checkbox" className="drawer-toggle" />
        <div className="drawer-content flex flex-col min-w-0">
            <header className="sticky top-0 z-30 lg:hidden flex items-center justify-between gap-2 px-4 pt-[env(safe-area-inset-top)] h-[calc(3.5rem+env(safe-area-inset-top))] bg-base-100/75 backdrop-blur-xl border-b border-white/5">
                <Logo/>
                <div className="flex items-center gap-1">
                    {isAdmin && <NavLink to="/notifications"
                        className={({ isActive }) => `btn btn-ghost btn-circle relative ${isActive ? "text-primary" : "text-base-content/70"}`}
                        aria-label={unread > 0 ? `Notifications, ${unread} unread` : "Notifications"}>
                        <Bell size={20}/>
                        {unread > 0 && <span className="absolute top-1 right-0.5 min-w-4 h-4 px-1 rounded-full bg-primary text-primary-content text-[10px] font-semibold grid place-items-center tabular-nums">
                            {badgeCount(unread)}
                        </span>}
                    </NavLink>}
                    <label htmlFor="my-drawer-3" className="btn btn-ghost btn-circle" aria-label="Open account menu">
                        <Avatar name={me?.username} className="size-8 text-sm"/>
                    </label>
                </div>
            </header>
            {/* the bottom padding keeps the tab bar and audio player (--player-h) off the end of the page */}
            <main className="w-full max-w-[1600px] mx-auto px-4 sm:px-6 lg:px-10 pt-5 sm:pt-6 lg:pt-8
                pb-[calc(6rem+var(--player-h,0px)+env(safe-area-inset-bottom))] lg:pb-[calc(2rem+var(--player-h,0px))]">
                {children}
            </main>
        </div>
        <Sidebar />
        <MobileNav />
    </div>
}
