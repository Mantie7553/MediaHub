import { getRole } from "../../utils/auth";
import LibrarySyncSection from "./LibrarySyncSection";
import PageHeader from "../../components/layout/PageHeader";

/**
 * Settings page. Admins also get server settings such as the library sync
 */
export default function SettingsPage() {
    const isAdmin = getRole() === "admin";

    return (
        <div className="flex flex-col gap-6">
            <PageHeader title="Settings" subtitle={isAdmin ? "Server and library settings." : "Your MediaHub settings."}/>
            {isAdmin && <LibrarySyncSection />}
        </div>
    );
}
