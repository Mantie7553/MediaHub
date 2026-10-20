import { getRole } from "../../utils/auth";
import AccountSection from "./AccountSection";
import LibrarySyncSection from "./LibrarySyncSection";
import PageHeader from "../../components/layout/PageHeader";

/**
 * Settings page: the user's account, and for admins server settings such as the library sync
 */
export default function SettingsPage() {
    const isAdmin = getRole() === "admin";

    return (
        <div>
            <PageHeader title="Settings" subtitle={isAdmin ? "Your account, and the server's library." : "Your MediaHub account."}/>
            <div className="grid gap-6 xl:grid-cols-[minmax(0,2fr)_minmax(0,3fr)] items-start">
                <AccountSection/>
                {isAdmin && <LibrarySyncSection />}
            </div>
        </div>
    );
}
