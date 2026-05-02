import Sidebar from "./Sidebar";

export default function Layout({children}) {
    return <div className="flex gap-2">
        <Sidebar/>
        <main className="w-full p-2">
            {children}
        </main>
    </div>
}