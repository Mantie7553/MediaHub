import Sidebar from "./Sidebar";

/**
 * Component to layout our pages properly, always puts the Sidebar on the left
 * @param {any} children the other page contents
 * @returns
 */
export default function Layout({children}) {
    return <div className="drawer lg:drawer-open">
        <input id="my-drawer-3" type="checkbox" className="drawer-toggle" />
        <div className="drawer-content flex flex-col">
            <div className="navbar bg-base-200 lg:hidden">
                <label htmlFor="my-drawer-3" className="btn btn-ghost">
                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" className="inline-block w-5 h-5 stroke-current">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 6h16M4 12h16M4 18h16"/>
                    </svg>
                </label>
                <h1 className="text-lg font-bold">Media<span className="text-primary">Hub</span></h1>
            </div>
            <main className="w-full p-2">
                {children}
            </main>
        </div>
        <Sidebar />
    </div>
}