import { Menu } from "lucide-react";
import Sidebar from "./Sidebar";
import Logo from "./Logo";
import useAudioStore from "../../stores/useAudioStore";

/**
 * Component to layout our pages properly, always puts the Sidebar on the left
 * @param {any} children the other page contents
 * @returns
 */
export default function Layout({children}) {
    // leave room so the audio player never covers the end of the page
    const playing = useAudioStore(state => state.currentTrack !== null);

    return <div className="drawer lg:drawer-open">
        <input id="my-drawer-3" type="checkbox" className="drawer-toggle" />
        <div className="drawer-content flex flex-col min-w-0">
            <div className="navbar sticky top-0 z-30 lg:hidden bg-base-100/75 backdrop-blur-lg border-b border-white/5 gap-2">
                <label htmlFor="my-drawer-3" className="btn btn-ghost btn-square" aria-label="Open navigation">
                    <Menu size={20}/>
                </label>
                <Logo/>
            </div>
            <main className={`w-full max-w-[1600px] mx-auto px-4 sm:px-6 lg:px-10 py-6 lg:py-8 ${playing ? "pb-28" : ""}`}>
                {children}
            </main>
        </div>
        <Sidebar />
    </div>
}
