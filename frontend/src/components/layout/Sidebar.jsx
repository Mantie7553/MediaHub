import { NavLink, useLocation, useNavigate } from "react-router-dom"
import api from "../../services/api"

/**
 *  Navigation for the application
 * @returns A sidebar used for navigating the application
 */
export default function Sidebar() {
    const navigate = useNavigate();

    function handleLogout() {
        api.delete("/auth/logout", {
            data: {refresh_token: localStorage.getItem("refresh_token")}
        });
        localStorage.removeItem("token");
        localStorage.removeItem("refresh_token");
        navigate("/login");
    }

    return <div className="drawer lg:drawer-open w-fit min-h-screen">
        <input id="my-drawer-3" type="checkbox" className="drawer-toggle" />
        <div className=" flex flex-col h-full drawer-side bg-base-200 p-2 items-center">
            <h1 className="text-lg font-bold">Media<span className="text-primary">Hub</span></h1>
            <ul className="menu flex-1">
                <NavItem path="/" title="Dashboard"/>
                <NavItem path="/downloads" title="Downloads"/>
                <NavItem path="/media" title="Media"/>
                <NavItem path="/manga" title="Manga" />
                <NavItem path="/settings" title="Settings"/>
            </ul>
            <button className="btn btn-ghost text-error justify-start" onClick={handleLogout}>Log Out</button>
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