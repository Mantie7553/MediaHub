import Sidebar, { Avatar } from "./Sidebar";
import Logo from "./Logo";
import MobileNav from "./MobileNav";
import useAudioStore from "../../stores/useAudioStore";
import useMe from "../../hooks/useMe";

/**
 * Component to layout our pages properly. The Sidebar is always on the left on desktop, phones
 * get a tab bar along the bottom and the sidebar as a drawer behind their avatar
 * @param {any} children the other page contents
 * @returns
 */
export default function Layout({children}) {
    const { me } = useMe();
    // leave room so the tab bar and audio player never cover the end of the page
    const playing = useAudioStore(state => state.currentTrack !== null);

    return <div className="drawer lg:drawer-open">
        <input id="my-drawer-3" type="checkbox" className="drawer-toggle" />
        <div className="drawer-content flex flex-col min-w-0">
            <header className="sticky top-0 z-30 lg:hidden flex items-center justify-between gap-2 px-4 pt-[env(safe-area-inset-top)] h-[calc(3.5rem+env(safe-area-inset-top))] bg-base-100/75 backdrop-blur-xl border-b border-white/5">
                <Logo/>
                <label htmlFor="my-drawer-3" className="btn btn-ghost btn-circle" aria-label="Open account menu">
                    <Avatar name={me?.username} className="size-8 text-sm"/>
                </label>
            </header>
            <main className={`w-full max-w-[1600px] mx-auto px-4 sm:px-6 lg:px-10 pt-5 sm:pt-6 lg:py-8
                ${playing ? "pb-[calc(10rem+env(safe-area-inset-bottom))] lg:pb-28" : "pb-[calc(6rem+env(safe-area-inset-bottom))] lg:pb-8"}`}>
                {children}
            </main>
        </div>
        <Sidebar />
        <MobileNav />
    </div>
}
