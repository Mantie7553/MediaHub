import Sidebar, { Avatar } from "./Sidebar";
import Logo from "./Logo";
import MobileNav from "./MobileNav";
import useMe from "../../hooks/useMe";

/**
 * Component to layout our pages properly. The Sidebar is always on the left on desktop, phones
 * get a tab bar along the bottom and the sidebar as a drawer behind their avatar
 * @param {any} children the other page contents
 * @returns
 */
export default function Layout({children}) {
    const { me } = useMe();

    return <div className="drawer lg:drawer-open">
        <input id="my-drawer-3" type="checkbox" className="drawer-toggle" />
        <div className="drawer-content flex flex-col min-w-0">
            <header className="sticky top-0 z-30 lg:hidden flex items-center justify-between gap-2 px-4 pt-[env(safe-area-inset-top)] h-[calc(3.5rem+env(safe-area-inset-top))] bg-base-100/75 backdrop-blur-xl border-b border-white/5">
                <Logo/>
                <label htmlFor="my-drawer-3" className="btn btn-ghost btn-circle" aria-label="Open account menu">
                    <Avatar name={me?.username} className="size-8 text-sm"/>
                </label>
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
