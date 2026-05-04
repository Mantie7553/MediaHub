import Sidebar from "./Sidebar";

/**
 * Component to layout our pages properly, always puts the Sidebar on the left
 * @param {any} children the other page contents
 * @returns
 */
export default function Layout({children}) {
    return <div className="flex gap-2">
        <Sidebar/>
        <main className="w-full p-2">
            {children}
        </main>
    </div>
}