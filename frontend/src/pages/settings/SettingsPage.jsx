import { useNavigate } from "react-router-dom";
import { LogOut, UserRound } from "lucide-react";
import { getRole, logout } from "../../utils/auth";
import LibrarySyncSection from "./LibrarySyncSection";
import PageHeader from "../../components/layout/PageHeader";
import { Panel } from "../../components/DataTable";
import { Avatar } from "../../components/layout/Sidebar";
import StatusPill from "../../components/StatusPill";
import useMe from "../../hooks/useMe";

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

/**
 * Who is signed in, with a way to sign out
 */
function AccountSection() {
    const navigate = useNavigate();
    const { me, loading } = useMe();

    return <Panel title="Account" icon={UserRound}>
        <div className="flex items-center gap-4">
            <Avatar name={me?.username} className="size-14 text-xl"/>
            <div className="min-w-0 flex-1">
                {loading ? <>
                    <div className="skeleton h-4 w-32 mb-2"></div>
                    <div className="skeleton h-3 w-44"></div>
                </> : <>
                    <p className="font-semibold text-lg truncate">{me?.username}</p>
                    <p className="text-sm text-base-content/55 truncate">{me?.email}</p>
                </>}
            </div>
            {me?.role && <StatusPill label={me.role} tone={me.role === "admin" ? "primary" : "secondary"}/>}
        </div>
        <div className="flex justify-end border-t border-white/5 mt-5 pt-4">
            <button className="btn btn-ghost text-error hover:bg-error/10" onClick={() => logout(navigate)}>
                <LogOut size={16}/> Log out
            </button>
        </div>
    </Panel>
}
