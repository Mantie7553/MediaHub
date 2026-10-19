import { NavLink, useNavigate } from "react-router-dom"
import { Compass, Download, LayoutDashboard, LogOut, PanelLeftClose, PanelLeftOpen, Settings, Users } from "lucide-react"
import api from "../../services/api"
import { getRole } from "../../utils/auth"
import useMe from "../../hooks/useMe"
import useSidebarStore from "../../stores/useSidebarStore"
import Logo from "./Logo"

/**
 *  Navigation for the application. On desktop it can shrink to an icon rail
 * @returns A sidebar used for navigating the application
 */
export default function Sidebar() {
    const navigate = useNavigate();
    const { me } = useMe();
    const role = me?.role ?? getRole();
    const collapsed = useSidebarStore(state => state.collapsed);
    const toggle = useSidebarStore(state => state.toggle);

    function handleLogout() {
        api.delete("/auth/logout", {
            data: {refresh_token: localStorage.getItem("refresh_token")}
        });
        localStorage.removeItem("token");
        localStorage.removeItem("refresh_token");
        navigate("/login");
    }

    // the rail only applies on desktop, the mobile drawer always shows labels
    const hideInRail = collapsed ? "lg:hidden" : "";

    return <div className="drawer-side z-40">
        <label htmlFor="my-drawer-3" className="drawer-overlay"></label>
        <aside className={`flex flex-col h-full bg-base-200 border-r border-white/5 py-4 transition-[width] duration-200 ease-out w-64 ${collapsed ? "lg:w-[4.5rem]" : "lg:w-60"}`}>
            <div className={`px-4 mb-6 ${collapsed ? "lg:px-0 lg:flex lg:justify-center" : ""}`}>
                {collapsed && <span className="hidden lg:inline-flex"><Logo showName={false}/></span>}
                <span className={hideInRail}><Logo/></span>
            </div>

            <nav className="flex-1 flex flex-col gap-1 px-3 overflow-y-auto overflow-x-hidden">
                <NavItem path="/" title="Dashboard" icon={LayoutDashboard} collapsed={collapsed}/>
                <NavItem path="/discover" title="Discover" icon={Compass} collapsed={collapsed}/>
                {role === "admin" && <>
                    <p className={`px-3 pt-5 pb-1 text-[11px] font-medium uppercase tracking-wider text-base-content/40 ${hideInRail}`}>Admin</p>
                    {collapsed && <div className="hidden lg:block mx-2 my-3 border-t border-white/5"></div>}
                    <NavItem path="/downloads" title="Downloads" icon={Download} collapsed={collapsed}/>
                    <NavItem path="/users" title="Users" icon={Users} collapsed={collapsed}/>
                </>}
                <div className="mt-auto pt-4">
                    <NavItem path="/settings" title="Settings" icon={Settings} collapsed={collapsed}/>
                </div>
            </nav>

            <div className="px-3 pt-3 mt-3 border-t border-white/5 flex flex-col gap-1">
                <div className={`flex items-center gap-3 p-1.5 ${collapsed ? "lg:flex-col lg:px-0 lg:gap-2" : ""}`}>
                    <span className="brand-gradient size-9 shrink-0 rounded-full grid place-items-center text-sm font-semibold text-white uppercase" aria-hidden="true">
                        {(me?.username ?? "").charAt(0)}
                    </span>
                    <div className={`min-w-0 flex-1 ${hideInRail}`}>
                        <p className="text-sm font-medium truncate">{me?.username ?? " "}</p>
                        <p className="text-xs text-base-content/50 capitalize">{role ?? ""}</p>
                    </div>
                    <span className={collapsed ? "lg:tooltip lg:tooltip-right" : ""} data-tip="Log out">
                        <button className="btn btn-ghost btn-sm btn-square text-base-content/60 hover:text-error" onClick={handleLogout} aria-label="Log out">
                            <LogOut size={18}/>
                        </button>
                    </span>
                </div>
                <button
                    className={`hidden lg:flex btn btn-ghost btn-sm gap-3 text-base-content/50 font-normal ${collapsed ? "justify-center px-0" : "justify-start"}`}
                    onClick={toggle}
                    aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
                >
                    {collapsed ? <PanelLeftOpen size={18}/> : <><PanelLeftClose size={18}/> Collapse</>}
                </button>
            </div>
        </aside>
    </div>
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
        <span className={`block ${collapsed ? "lg:tooltip lg:tooltip-right" : ""}`} data-tip={title}>
            <NavLink
                to={path}
                end={path === "/"}
                aria-label={title}
                className={({ isActive }) => [
                    "relative flex items-center gap-3 h-10 px-3 rounded-field text-sm transition-colors duration-150",
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
        </span>
    )
}
