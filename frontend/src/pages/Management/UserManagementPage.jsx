import { forwardRef, useRef, useState, useEffect } from "react";
import api from "../../services/api";
import Format from "../../utils/format";
import Loading from "../../components/states/Loading";
import Error from "../../components/states/Error";

/**
 * Admin page for managing users
 */
export default function UserManagementPage() {
    const [users, setUsers] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");
    const [selectedUser, setSelectedUser] = useState(null);
    const [deleteTarget, setDeleteTarget] = useState(null);
    const modalRef = useRef(null);
    const deleteModalRef = useRef(null);

    function fetchUsers() {
        setLoading(true);
        api.get("/admin/users")
            .then(res => setUsers(res.data ?? []))
            .catch(err => setError(err.message))
            .finally(() => setLoading(false));
    }

    useEffect(() => { fetchUsers(); }, []);

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
            .then(() => fetchUsers())
            .catch(err => setError(err.response?.data?.error ?? err.message))
            .finally(() => deleteModalRef.current.close());
    }

    function handleSave(data) {
        const req = selectedUser
            ? api.put(`/admin/users/${selectedUser.id}`, data)
            : api.post("/admin/users", data);

        return req.then(() => fetchUsers());
    }

    if (loading) return <Loading />;
    if (error) return <Error error={error} />;

    return (
        <div className="flex flex-col gap-6">
            <div className="flex items-center justify-between">
                <h1>User Management</h1>
                <button className="btn btn-primary btn-sm" onClick={handleAdd}>Add User</button>
            </div>
            <section className="card">
                <Table
                    columns={[
                        { key: "username", header: "Username" },
                        { key: "email", header: "Email" },
                        { key: "role", header: "Role", render: (val) => Format.cleanString(val) },
                        { key: "download_permission", header: "Download Permission", render: (val) => Format.cleanString(val) },
                        { key: "created_at", header: "Joined", render: (val) => Format.date(val) },
                        { key: "actions", header: "Actions", render: (_, item) => (
                            <div className="flex gap-2">
                                <button className="btn btn-xs btn-outline" onClick={() => handleEdit(item)}>Edit</button>
                                <button className="btn btn-xs btn-error" onClick={() => handleDeleteClick(item)}>Delete</button>
                            </div>
                        )},
                    ]}
                    items={users}
                />
            </section>

            <UserModal ref={modalRef} user={selectedUser} onSave={handleSave} />
            <DeleteModal ref={deleteModalRef} user={deleteTarget} onConfirm={handleDeleteConfirm} />
        </div>
    );
}

/**
 * Builds a table with appropriate column headings and fields
 */
function Table({ columns, items }) {
    return (
        <table className="table">
            <thead>
                <tr>
                    {columns.map(col => <th key={col.key}>{col.header}</th>)}
                </tr>
            </thead>
            <tbody>
                {items.map((item, i) => (
                    <tr key={i}>
                        {columns.map(col => (
                            <td key={col.key}>
                                {col.render ? col.render(item[col.key], item) : item[col.key]}
                            </td>
                        ))}
                    </tr>
                ))}
            </tbody>
        </table>
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

        const payload = isEdit
            ? { role, download_permission: downloadPermission, ...(password ? { password } : {}) }
            : { username, email, password, role, download_permission: downloadPermission };

        setLoading(true);
        onSave(payload)
            .then(() => ref.current.close())
            .catch(err => setError(err.response?.data ?? err.message))
            .finally(() => setLoading(false));
    }

    const isEdit = !!user;

    return (
        <dialog className="modal" ref={ref}>
            <div className="modal-box flex flex-col gap-4">
                <h3 className="font-bold text-lg">{isEdit ? `Edit ${user.username}` : "Add User"}</h3>

                {!isEdit && <>
                    <label className="flex flex-col gap-1">
                        <span className="text-sm">Username</span>
                        <input className="input input-bordered" value={username} onChange={e => setUsername(e.target.value)} />
                    </label>
                    <label className="flex flex-col gap-1">
                        <span className="text-sm">Email</span>
                        <input className="input input-bordered" type="email" value={email} onChange={e => setEmail(e.target.value)} />
                    </label>
                </>}

                <label className="flex flex-col gap-1">
                    <span className="text-sm">{isEdit ? "New Password (leave blank to keep current)" : "Password"}</span>
                    <input className="input input-bordered" type="password" value={password} onChange={e => setPassword(e.target.value)} />
                </label>

                <label className="flex flex-col gap-1">
                    <span className="text-sm">Role</span>
                    <select className="select select-bordered" value={role} onChange={e => setRole(e.target.value)}>
                        <option value="user">User</option>
                        <option value="admin">Admin</option>
                    </select>
                </label>

                <label className="flex flex-col gap-1">
                    <span className="text-sm">Download Permission</span>
                    <select className="select select-bordered" value={downloadPermission} onChange={e => setDownloadPermission(e.target.value)}>
                        <option value="vetted">Vetted</option>
                        <option value="auto_approved">Auto Approved</option>
                    </select>
                </label>

                {error && <p className="text-error text-sm">{error}</p>}

                <div className="flex gap-2 justify-end">
                    <button className="btn btn-primary" onClick={handleConfirm} disabled={loading}>
                        {isEdit ? "Save" : "Create"}
                    </button>
                    <button className="btn btn-ghost" onClick={() => ref.current.close()}>Cancel</button>
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
        <dialog className="modal" ref={ref}>
            <div className="modal-box flex flex-col gap-4">
                <h3 className="font-bold text-lg">Delete User</h3>
                <p>Are you sure you want to delete <span className="font-semibold">{user?.username}</span>? This cannot be undone.</p>
                <div className="flex gap-2 justify-end">
                    <button className="btn btn-error" onClick={onConfirm}>Delete</button>
                    <button className="btn btn-ghost" onClick={() => ref.current.close()}>Cancel</button>
                </div>
            </div>
        </dialog>
    );
});