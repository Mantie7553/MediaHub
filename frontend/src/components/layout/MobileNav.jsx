import { NavLink } from "react-router-dom"
import { getRole } from "../../utils/auth"
import useMe from "../../hooks/useMe"
import navItems from "./navItems"

/**
 * The tab bar along the bottom of the screen on phones and tablets, in place of the sidebar
 */
export default function MobileNav() {
    const { me } = useMe();
    const items = navItems(me?.role ?? getRole());

    return <nav className="lg:hidden fixed inset-x-0 bottom-0 z-30 bg-base-200/85 backdrop-blur-xl border-t border-white/5 pb-[env(safe-area-inset-bottom)]"
        aria-label="Main">
        <ul className="flex h-16 max-w-xl mx-auto">
            {items.map(({ path, short, icon: Icon }) => (
                <li key={path} className="flex-1">
                    <NavLink
                        to={path}
                        end={path === "/"}
                        className={({ isActive }) => `relative h-full flex flex-col items-center justify-center gap-1 text-[11px] transition-colors
                            ${isActive ? "text-primary font-medium" : "text-base-content/55 active:text-base-content"}`}
                    >
                        {({ isActive }) => <>
                            {isActive && <span className="absolute top-0 h-[3px] w-8 rounded-b-full bg-primary shadow-[0_0_12px_2px_rgb(242_89_18/0.6)]"></span>}
                            <span className={`grid place-items-center h-7 w-12 rounded-full transition-colors ${isActive ? "bg-primary/12" : ""}`}>
                                <Icon size={20} strokeWidth={isActive ? 2.25 : 1.75}/>
                            </span>
                            {short}
                        </>}
                    </NavLink>
                </li>
            ))}
        </ul>
    </nav>
}
