import { NavLink, useLocation } from "react-router-dom"

/**
 *  Navigation for the application
 * @returns A sidebar used for navigating the application
 */
export default function Sidebar() {
    return <div className="drawer lg:drawer-open w-fit">
        <input id="my-drawer-3" type="checkbox" className="drawer-toggle" />
        <div className="drawer-side bg-base-200 p-2">
            <h1 className="text-lg font-bold">Media<span className="text-primary">Hub</span></h1>
            <ul className="menu">
                <NavItem path="/" title="Dashboard"/>
                <NavItem path="/downloads" title="Downloads"/>
                <NavItem path="/media" title="Media"/>
                <NavItem path="/settings" title="Settings"/>
            </ul>
        </div>
    </div>
}

/**
 * A navigation Item, used for navigation links in the Sidebar
 * @param path the path the link will go to
 * @param title the name displayed on the link
 * @returns a li with the NavLink to some page
 */
function NavItem({path, title}) {
    return (
        <li>
            <NavLink 
                to={path}
                className={({ isActive }) => isActive ? "text-primary font-semibold" : ""}
            >
                {title}
            </NavLink>
        </li>
    )
}