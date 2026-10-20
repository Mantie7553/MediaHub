import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { KeyRound, LogOut, Mail, MonitorSmartphone, Pencil, User, UserRound } from "lucide-react";
import api from "../../services/api";
import { logout } from "../../utils/auth";
import { errorMessage } from "../../utils/errors";
import { validatePassword } from "../../utils/validate";
import { toast } from "../../stores/useToastStore";
import { Panel } from "../../components/DataTable";
import { Avatar } from "../../components/layout/Sidebar";
import StatusPill from "../../components/StatusPill";
import TextField from "../../components/TextField";
import useMe from "../../hooks/useMe";

/**
 * Who is signed in, with forms to change their name, email and password, and ways to sign out
 * here or everywhere
 */
export default function AccountSection() {
    const navigate = useNavigate();
    const { me, loading, refresh } = useMe();
    // which form is open: "profile", "password" or null
    const [editing, setEditing] = useState(null);

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

        {editing === "profile" && me && <ProfileForm me={me} onDone={changed => { setEditing(null); if (changed) refresh(); }}/>}
        {editing === "password" && <PasswordForm onDone={() => setEditing(null)}/>}

        {!editing && <div className="flex flex-wrap gap-2 mt-5">
            <button className="btn btn-sm btn-ghost border-white/10 bg-white/5 hover:bg-white/10 gap-1.5" onClick={() => setEditing("profile")} disabled={!me}>
                <Pencil size={14}/> Edit name or email
            </button>
            <button className="btn btn-sm btn-ghost border-white/10 bg-white/5 hover:bg-white/10 gap-1.5" onClick={() => setEditing("password")}>
                <KeyRound size={14}/> Change password
            </button>
        </div>}

        <div className="flex flex-wrap justify-end gap-2 border-t border-white/5 mt-5 pt-4">
            <button className="btn btn-ghost text-base-content/70" onClick={() => logout(navigate, true)}
                title="Ends your sessions on every device, including this one">
                <MonitorSmartphone size={16}/> Log out everywhere
            </button>
            <button className="btn btn-ghost text-error hover:bg-error/10" onClick={() => logout(navigate)}>
                <LogOut size={16}/> Log out
            </button>
        </div>
    </Panel>
}

/**
 * Change the username and/or email, confirmed with the current password
 * @param {object} me the signed in user
 * @param {function} onDone called with whether anything changed
 */
function ProfileForm({ me, onDone }) {
    const [username, setUsername] = useState(me.username);
    const [email, setEmail] = useState(me.email);
    const [currentPassword, setCurrentPassword] = useState("");
    const [error, setError] = useState(null);
    const [saving, setSaving] = useState(false);
    const unchanged = username.trim() === me.username && email.trim().toLowerCase() === me.email;

    function handleSubmit(e) {
        e.preventDefault();
        if (unchanged) { onDone(false); return; }
        setSaving(true);
        setError(null);
        api.put("/me", {
            username: username.trim() === me.username ? "" : username.trim(),
            email: email.trim().toLowerCase() === me.email ? "" : email.trim(),
            current_password: currentPassword,
        })
            .then(() => { toast.success("Account updated"); onDone(true); })
            .catch(err => setError(errorMessage(err, "Couldn't update your account")))
            .finally(() => setSaving(false));
    }

    return <form className="flex flex-col gap-3 mt-5 border-t border-white/5 pt-5" onSubmit={handleSubmit}>
        <TextField label="Username" icon={User} value={username} onChange={e => setUsername(e.target.value)} autoComplete="username" required/>
        <TextField label="Email" icon={Mail} type="email" value={email} onChange={e => setEmail(e.target.value)} autoComplete="email" required/>
        <TextField label="Current password" icon={KeyRound} type="password" value={currentPassword}
            onChange={e => setCurrentPassword(e.target.value)} autoComplete="current-password" required
            hint="Confirms it's you making the change."/>
        {error && <p className="text-sm text-error" role="alert">{error}</p>}
        <div className="flex justify-end gap-2">
            <button type="button" className="btn btn-ghost" onClick={() => onDone(false)}>Cancel</button>
            <button type="submit" className="btn btn-primary" disabled={saving || unchanged || !currentPassword}>
                {saving && <span className="loading loading-spinner loading-xs"/>} Save
            </button>
        </div>
    </form>
}

/**
 * Change the password. The server signs out every other session
 * @param {function} onDone called when the form closes
 */
function PasswordForm({ onDone }) {
    const [currentPassword, setCurrentPassword] = useState("");
    const [newPassword, setNewPassword] = useState("");
    const [confirm, setConfirm] = useState("");
    const [error, setError] = useState(null);
    const [saving, setSaving] = useState(false);

    function handleSubmit(e) {
        e.preventDefault();
        const problem = validatePassword(newPassword) ?? (confirm !== newPassword ? "The new passwords don't match" : null);
        if (problem) { setError(problem); return; }
        setSaving(true);
        setError(null);
        api.put("/me/password", { current_password: currentPassword, new_password: newPassword })
            .then(() => { toast.success("Password changed. Other devices have been signed out"); onDone(); })
            .catch(err => setError(errorMessage(err, "Couldn't change your password")))
            .finally(() => setSaving(false));
    }

    return <form className="flex flex-col gap-3 mt-5 border-t border-white/5 pt-5" onSubmit={handleSubmit}>
        <TextField label="Current password" icon={KeyRound} type="password" value={currentPassword}
            onChange={e => setCurrentPassword(e.target.value)} autoComplete="current-password" required/>
        <TextField label="New password" icon={KeyRound} type="password" value={newPassword}
            onChange={e => setNewPassword(e.target.value)} autoComplete="new-password" required
            hint="At least 12 characters with upper and lower case, a number and a symbol."/>
        <TextField label="Confirm new password" icon={KeyRound} type="password" value={confirm}
            onChange={e => setConfirm(e.target.value)} autoComplete="new-password" required/>
        {error && <p className="text-sm text-error" role="alert">{error}</p>}
        <div className="flex justify-end gap-2">
            <button type="button" className="btn btn-ghost" onClick={onDone}>Cancel</button>
            <button type="submit" className="btn btn-primary" disabled={saving || !currentPassword || !newPassword}>
                {saving && <span className="loading loading-spinner loading-xs"/>} Change password
            </button>
        </div>
    </form>
}
