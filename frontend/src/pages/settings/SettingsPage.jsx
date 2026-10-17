import { getRole } from "../../utils/auth";
import LibrarySyncSection from "./LibrarySyncSection";

/**
 * Settings page. Admins also get server settings such as the library sync
 */
export default function SettingsPage() {
    const isAdmin = getRole() === "admin";

    return (
        <div className="flex flex-col gap-6">
            <h1>Settings</h1>
            {isAdmin && <LibrarySyncSection />}
        </div>
    );
}
