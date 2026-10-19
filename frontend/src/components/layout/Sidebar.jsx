import { NavLink, useNavigate } from "react-router-dom"
import { LogOut, PanelLeftClose, PanelLeftOpen } from "lucide-react"
import { getRole, logout } from "../../utils/auth"
import useMe from "../../hooks/useMe"
import useSidebarStore from "../../stores/useSidebarStore"
import Logo from "./Logo"
import RailTooltip from "./RailTooltip"
import navItems from "./navItems"

/**
 *  Navigation for the application. On desktop it can shrink to an icon rail, on mobile it is
 *  the drawer behind the avatar in the top bar
 * @returns A sidebar used for navigating the application
 */
export default function Sidebar() {
    const navigate = useNavigate();
    const { me } = useMe();
    const role = me?.role ?? getRole();
    const collapsed = useSidebarStore(state => state.collapsed);
    const toggle = useSidebarStore(state => state.toggle);
    const items = navItems(role);

    // the rail only applies on desktop, the mobile drawer always shows labels
    const hideInRail = collapsed ? "lg:hidden" : "";
    const navItem = item => <NavItem key={item.path} {...item} collapsed={collapsed}/>;

    return <div className="drawer-side z-40">
        <label htmlFor="my-drawer-3" className="drawer-overlay"></label>
        <aside className={`flex flex-col h-full bg-base-200 border-r border-white/5 pt-[calc(1rem+env(safe-area-inset-top))] pb-[calc(1rem+env(safe-area-inset-bottom))] transition-[width] duration-200 ease-out w-72 max-w-[85vw] ${collapsed ? "lg:w-[4.5rem]" : "lg:w-60"}`}>
            <div className={`px-4 mb-6 ${collapsed ? "lg:px-0 lg:flex lg:justify-center" : ""}`}>
                {collapsed && <span className="hidden lg:inline-flex"><Logo showName={false}/></span>}
                <span className={hideInRail}><Logo/></span>
            </div>

            <nav className="flex-1 flex flex-col gap-1 px-3 overflow-y-auto overflow-x-hidden">
                {items.filter(item => item.group === "main").map(navItem)}
                {role === "admin" && <>
                    <p className={`px-3 pt-5 pb-1 text-[11px] font-medium uppercase tracking-wider text-base-content/40 ${hideInRail}`}>Admin</p>
                    {collapsed && <div className="hidden lg:block mx-2 my-3 border-t border-white/5"></div>}
                    {items.filter(item => item.group === "admin").map(navItem)}
                </>}
                <div className="mt-auto pt-4">
                    {items.filter(item => item.group === "footer").map(navItem)}
                </div>
            </nav>

            <div className="px-3 pt-3 mt-3 border-t border-white/5 flex flex-col gap-1">
                <div className={`flex items-center gap-3 p-1.5 ${collapsed ? "lg:flex-col lg:px-0 lg:gap-2" : ""}`}>
                    <Avatar name={me?.username}/>
                    <div className={`min-w-0 flex-1 ${hideInRail}`}>
                        <p className="text-sm font-medium truncate">{me?.username ?? " "}</p>
                        <p className="text-xs text-base-content/50 capitalize">{role ?? ""}</p>
                    </div>
                    <RailTooltip label="Log out" enabled={collapsed}>
                        <button className="btn btn-ghost btn-square text-base-content/60 hover:text-error" onClick={() => logout(navigate)} aria-label="Log out">
                            <LogOut size={18}/>
                        </button>
                    </RailTooltip>
                </div>
                <RailTooltip label="Expand sidebar" enabled={collapsed}>
                    <button
                        className={`hidden lg:flex w-full btn btn-ghost btn-sm gap-3 text-base-content/50 font-normal ${collapsed ? "justify-center px-0" : "justify-start"}`}
                        onClick={toggle}
                        aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
                    >
                        {collapsed ? <PanelLeftOpen size={18}/> : <><PanelLeftClose size={18}/> Collapse</>}
                    </button>
                </RailTooltip>
            </div>
        </aside>
    </div>
}

/**
 * The user's initial in a gradient circle
 * @param {string} name the username, empty while it loads
 * @param {string} className size overrides
 */
export function Avatar({ name, className = "size-9 text-sm" }) {
    return <span className={`brand-gradient shrink-0 rounded-full grid place-items-center font-semibold text-white uppercase ${className}`} aria-hidden="true">
        {(name ?? "").charAt(0)}
    </span>
}

/**
 * A navigation Item, used for navigation links in the Sidebar
 * @param path the path the link will go to
 * @param title the name displayed on the link
 * @param icon a lucide icon component
 * @param collapsed the sidebar is an icon rail, the title shows as a tooltip
 * @returns a NavLink to some page
 */
function NavItem({path, title, icon: Icon, collapsed}) {
    return (
        <RailTooltip label={title} enabled={collapsed}>
            <NavLink
                to={path}
                end={path === "/"}
                aria-label={title}
                // closes the mobile drawer after picking a page
                onClick={() => { const toggle = document.getElementById("my-drawer-3"); if (toggle) toggle.checked = false; }}
                className={({ isActive }) => [
                    "relative flex items-center gap-3 h-11 lg:h-10 px-3 rounded-field text-sm transition-colors duration-150",
                    collapsed ? "lg:justify-center lg:px-0" : "",
                    isActive
                        ? "bg-primary/12 text-primary font-medium"
                        : "text-base-content/70 hover:text-base-content hover:bg-white/5",
                ].join(" ")}
            >
                {({ isActive }) => <>
                    {isActive && <span className="absolute -left-3 top-2 bottom-2 w-[3px] rounded-r-full bg-primary shadow-[0_0_12px_2px_rgb(242_89_18/0.6)]"></span>}
                    <Icon size={19} strokeWidth={isActive ? 2.25 : 1.75} className="shrink-0"/>
                    <span className={collapsed ? "lg:hidden" : ""}>{title}</span>
                </>}
            </NavLink>
        </RailTooltip>
    )
}
