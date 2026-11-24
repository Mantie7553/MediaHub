import { userIsAdmin } from "../../utils/auth";
import AccountSection from "./AccountSection";
import PlaybackSection from "./PlaybackSection";
import HomePageSection from "./HomePageSection";
import LibrarySyncSection from "./LibrarySyncSection";
import AnimeMergeSection from "./AnimeMergeSection";
import SeriesLinksSection from "./SeriesLinksSection";
import PageHeader from "../../components/layout/PageHeader";

/**
 * Settings page: the user's account, home page and playback, and for admins server settings such as the library sync
 */
export default function SettingsPage() {
    const isAdmin = userIsAdmin();

    return (
        <div>
            <PageHeader title="Settings" subtitle={isAdmin ? "Your account, and the server's library." : "Your MediaHub account."}/>
            <div className="grid gap-6 xl:grid-cols-[minmax(0,2fr)_minmax(0,3fr)] items-start">
                <div className="grid gap-6">
                    <AccountSection/>
                    <HomePageSection/>
                    <PlaybackSection/>
                </div>
                {isAdmin && <div className="grid gap-6">
                    <LibrarySyncSection />
                    <AnimeMergeSection />
                    <SeriesLinksSection />
                </div>}
            </div>
        </div>
    );
}
