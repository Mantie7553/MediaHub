import PageHeader from "../../components/layout/PageHeader";
import { KeyRound, Mail, Pencil, Trash2, User, UserPlus, Users } from "lucide-react";
import { forwardRef, useRef, useState, useEffect } from "react";
import api from "../../services/api";
import useApi from "../../hooks/useApi";
import Format from "../../utils/format";
import { errorMessage } from "../../utils/errors";
import { toast } from "../../stores/useToastStore";
import ErrorState from "../../components/states/ErrorState";
import { TableSkeleton } from "../../components/states/Skeletons";
import { validatePassword } from "../../utils/validate";
import InvitesSection from "./InvitesSection";
import DataTable, { Panel } from "../../components/DataTable";
import StatusPill from "../../components/StatusPill";
import TextField, { SelectField } from "../../components/TextField";
import { Avatar } from "../../components/layout/Sidebar";

/**
 * Admin page for managing users
 */
export default function UserManagementPage() {
    const { data: users, loading, error, reload } = useApi("/admin/users", { initial: [] });
    const [selectedUser, setSelectedUser] = useState(null);
    const [deleteTarget, setDeleteTarget] = useState(null);
    const modalRef = useRef(null);
    const deleteModalRef = useRef(null);


    function handleAdd() {
        setSelectedUser(null);
        modalRef.current.showModal();
    }

    function handleEdit(user) {
        setSelectedUser(user);
        modalRef.current.showModal();
    }

    function handleDeleteClick(user) {
        setDeleteTarget(user);
        deleteModalRef.current.showModal();
    }

    function handleDeleteConfirm() {
        api.delete(`/admin/users/${deleteTarget.id}`)
            .then(() => { toast.success(`Deleted ${deleteTarget.username}`); reload(true); })
            .catch(err => toast.error(err, "Could not delete the user"))
            .finally(() => deleteModalRef.current.close());
    }

    function handleSave(data) {
        const req = selectedUser
            ? api.put(`/admin/users/${selectedUser.id}`, data)
            : api.post("/admin/users", data);

        return req.then(() => { toast.success(selectedUser ? "User updated" : "User added"); reload(true); });
    }

    return (
        <div className="flex flex-col gap-6">
            <PageHeader
                title="Users"
                subtitle="Manage who can sign in and what they're allowed to download."
                actions={<button className="btn btn-primary" onClick={handleAdd}><UserPlus size={16}/> Add user</button>}
            />
            <Panel title="People" icon={Users} meta={!loading && !error ? users.length : null}>
                {loading ? <TableSkeleton columns={5} rows={3} /> :
                error ? <ErrorState compact error={error} onRetry={reload} /> :
                <DataTable
                    items={users}
                    columns={[
                        { key: "username", header: "User", primary: true, render: (val, item) => (
                            <div className="flex items-center gap-3 min-w-0">
                                <Avatar name={val}/>
                                <div className="min-w-0">
                                    <p className="font-medium truncate">{val}</p>
                                    <p className="text-xs text-base-content/45 truncate">{item.email}</p>
                                </div>
                            </div>
                        )},
                        { key: "role", header: "Role", render: (val) => <StatusPill label={Format.cleanString(val)} tone={val === "admin" ? "primary" : "secondary"}/> },
                        { key: "download_permission", header: "Downloads", render: (val) => (
                            <StatusPill label={val === "auto_approved" ? "Auto-approved" : Format.cleanString(val)} tone={val === "auto_approved" ? "success" : "warning"}/>
                        )},
                        { key: "created_at", header: "Joined", render: (val) => <span className="text-base-content/60 whitespace-nowrap">{Format.date(val)}</span> },
                        { key: "actions", header: "", render: (_, item) => (
                            <div className="inline-flex gap-1">
                                <button className="btn btn-sm btn-ghost" onClick={() => handleEdit(item)} aria-label={`Edit ${item.username}`}>
                                    <Pencil size={15}/> <span className="md:hidden">Edit</span>
                                </button>
                                <button className="btn btn-sm btn-ghost text-error hover:bg-error/10" onClick={() => handleDeleteClick(item)} aria-label={`Delete ${item.username}`}>
                                    <Trash2 size={15}/> <span className="md:hidden">Delete</span>
                                </button>
                            </div>
                        )},
                    ]}
                />}
            </Panel>

            <InvitesSection />

            <UserModal ref={modalRef} user={selectedUser} onSave={handleSave} />
            <DeleteModal ref={deleteModalRef} user={deleteTarget} onConfirm={handleDeleteConfirm} />
        </div>
    );
}

/**
 * Modal for creating or editing a user
 */
const UserModal = forwardRef(function UserModal({ user, onSave }, ref) {
    const [username, setUsername] = useState("");
    const [email, setEmail] = useState("");
    const [password, setPassword] = useState("");
    const [role, setRole] = useState("user");
    const [downloadPermission, setDownloadPermission] = useState("vetted");
    const [error, setError] = useState("");
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        setUsername(user?.username ?? "");
        setEmail(user?.email ?? "");
        setPassword("");
        setRole(user?.role ?? "user");
        setDownloadPermission(user?.download_permission ?? "vetted");
        setError("");
    }, [user]);

    function handleConfirm() {
        setError("");
        const isEdit = !!user;

        if (!isEdit && (!username || !email || !password)) {
            setError("Username, email, and password are required.");
            return;
        }

        if (password) {
            const passwordError = validatePassword(password);
            if (passwordError) { setError(passwordError); return; }
        }

        setLoading(true);
        const payload = isEdit
            ? { role, download_permission: downloadPermission, ...(password ? { password } : {}) }
            : { username, email, password, role, download_permission: downloadPermission };

        onSave(payload)
            .then(() => ref.current.close())
            .catch(err => setError(errorMessage(err, "Could not save the user")))
            .finally(() => setLoading(false));
    }

    const isEdit = !!user;

    return (
        <dialog className="modal modal-bottom sm:modal-middle" ref={ref}>
            <div className="modal-box flex flex-col gap-4">
                <div className="flex items-center gap-3">
                    {isEdit ? <Avatar name={user.username} className="size-10 text-base"/> : <span className="grid size-10 place-items-center rounded-full bg-primary/15 text-primary"><UserPlus size={18}/></span>}
                    <div>
                        <h3 className="text-xl font-semibold">{isEdit ? `Edit ${user.username}` : "Add user"}</h3>
                        {isEdit && <p className="text-sm text-base-content/50">{user.email}</p>}
                    </div>
                </div>

                {!isEdit && <>
                    <TextField label="Username" icon={User} value={username} onChange={e => setUsername(e.target.value)} autoComplete="off" />
                    <TextField label="Email" icon={Mail} type="email" value={email} onChange={e => setEmail(e.target.value)} autoComplete="off" />
                </>}

                <TextField
                    label={isEdit ? "New password" : "Password"}
                    hint={isEdit ? "Leave blank to keep the current password." : undefined}
                    icon={KeyRound}
                    type="password"
                    value={password}
                    onChange={e => setPassword(e.target.value)}
                    autoComplete="new-password"
                />

                <div className="grid sm:grid-cols-2 gap-4">
                    <SelectField label="Role" value={role} onChange={e => setRole(e.target.value)}>
                        <option value="user">User</option>
                        <option value="admin">Admin</option>
                    </SelectField>
                    <SelectField label="Downloads" value={downloadPermission} onChange={e => setDownloadPermission(e.target.value)}>
                        <option value="vetted">Vetted (needs approval)</option>
                        <option value="auto_approved">Auto-approved</option>
                    </SelectField>
                </div>

                {error && <p role="alert" className="text-error text-sm">{error}</p>}

                <div className="flex gap-2 justify-end pt-1">
                    <button className="btn btn-ghost" onClick={() => ref.current.close()}>Cancel</button>
                    <button className="btn btn-primary" onClick={handleConfirm} disabled={loading}>
                        {loading && <span className="loading loading-spinner loading-xs"/>}
                        {isEdit ? "Save" : "Create user"}
                    </button>
                </div>
            </div>
        </dialog>
    );
});

/**
 * Confirmation modal for deleting a user
 */
const DeleteModal = forwardRef(function DeleteModal({ user, onConfirm }, ref) {
    return (
        <dialog className="modal modal-bottom sm:modal-middle" ref={ref}>
            <div className="modal-box flex flex-col gap-4">
                <h3 className="text-xl font-semibold flex items-center gap-2">
                    <Trash2 size={20} className="text-error"/> Delete {user?.username}?
                </h3>
                <p className="text-sm text-base-content/70">They won't be able to sign in any more. This can't be undone.</p>
                <div className="flex gap-2 justify-end">
                    <button className="btn btn-ghost" onClick={() => ref.current.close()}>Cancel</button>
                    <button className="btn btn-error" onClick={onConfirm}>Delete user</button>
                </div>
            </div>
        </dialog>
    );
});