import { forwardRef, useRef, useState, useEffect } from "react";
import api from "../../services/api";
import Format from "../../utils/format";

const STATUS_BADGE = {
    active: "badge-success",
    used: "badge-neutral",
    expired: "badge-ghost",
};

/**
 * Admin section for creating, listing and revoking single use sign up codes
 */
export default function InvitesSection() {
    const [invites, setInvites] = useState([]);
    const [error, setError] = useState("");
    const createModalRef = useRef(null);

    function fetchInvites() {
        api.get("/admin/invites")
            .then(res => setInvites(res.data ?? []))
            .catch(err => setError(err.response?.data ?? err.message));
    }

    useEffect(() => { fetchInvites(); }, []);

    function handleRevoke(invite) {
        api.delete(`/admin/invites/${invite.id}`)
            .then(() => fetchInvites())
            .catch(err => setError(err.response?.data ?? err.message));
    }

    return (
        <section className="flex flex-col gap-4">
            <div className="flex items-center justify-between">
                <h2 className="font-bold">Invites</h2>
                <button className="btn btn-primary btn-sm" onClick={() => createModalRef.current.showModal()}>Create Invite</button>
            </div>
            {error && <p className="text-error text-sm">{error}</p>}
            {invites.length === 0 ? (
                <p className="text-sm opacity-50">No invites yet. New users need an invite code to sign up.</p>
            ) : (
                <div className="card overflow-x-auto">
                    <table className="table">
                        <thead>
                            <tr>
                                <th>Code</th>
                                <th>Note</th>
                                <th>Status</th>
                                <th>Expires</th>
                                <th>Used By</th>
                                <th>Created By</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            {invites.map(invite => (
                                <tr key={invite.id}>
                                    <td className="font-mono">…{invite.code_hint}</td>
                                    <td>{invite.note ?? ""}</td>
                                    <td><span className={`badge badge-sm ${STATUS_BADGE[invite.status] ?? ""}`}>{invite.status}</span></td>
                                    <td>{Format.dateTime(invite.expires_at)}</td>
                                    <td>{invite.used_by ?? ""}</td>
                                    <td>{invite.created_by ?? ""}</td>
                                    <td>
                                        {invite.status === "active" && (
                                            <button className="btn btn-xs btn-error" onClick={() => handleRevoke(invite)}>Revoke</button>
                                        )}
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}

            <CreateInviteModal ref={createModalRef} onCreated={fetchInvites} />
        </section>
    );
}

/**
 * Modal that creates an invite and shows the code once
 */
const CreateInviteModal = forwardRef(function CreateInviteModal({ onCreated }, ref) {
    const [note, setNote] = useState("");
    const [days, setDays] = useState(7);
    const [created, setCreated] = useState(null);
    const [copied, setCopied] = useState("");
    const [error, setError] = useState("");
    const [loading, setLoading] = useState(false);

    const link = created ? `${window.location.origin}/login?invite=${encodeURIComponent(created.code)}` : "";

    function reset() {
        setNote("");
        setDays(7);
        setCreated(null);
        setCopied("");
        setError("");
    }

    function handleCreate() {
        setError("");
        setLoading(true);
        api.post("/admin/invites", { note, expires_in_days: Number(days) })
            .then(res => {
                setCreated(res.data);
                onCreated();
            })
            .catch(err => setError(err.response?.data ?? err.message))
            .finally(() => setLoading(false));
    }

    function copy(text, what) {
        navigator.clipboard?.writeText(text)
            .then(() => setCopied(what))
            .catch(() => setCopied(""));
    }

    function close() {
        ref.current.close();
    }

    return (
        <dialog className="modal" ref={ref} onClose={reset}>
            <div className="modal-box flex flex-col gap-4">
                <h3 className="font-bold text-lg">Create Invite</h3>

                {!created ? <>
                    <label className="flex flex-col gap-1">
                        <span className="text-sm">Note (optional)</span>
                        <input className="input input-bordered" value={note} onChange={e => setNote(e.target.value)} placeholder="Who is this for?" />
                    </label>
                    <label className="flex flex-col gap-1">
                        <span className="text-sm">Expires after</span>
                        <select className="select select-bordered" value={days} onChange={e => setDays(e.target.value)}>
                            <option value={1}>1 day</option>
                            <option value={7}>7 days</option>
                            <option value={30}>30 days</option>
                        </select>
                    </label>

                    {error && <p className="text-error text-sm">{error}</p>}

                    <div className="flex gap-2 justify-end">
                        <button className="btn btn-primary" onClick={handleCreate} disabled={loading}>Create</button>
                        <button className="btn btn-ghost" onClick={close}>Cancel</button>
                    </div>
                </> : <>
                    <p className="text-sm">
                        Share this code or link. It works for one sign up and expires {Format.dateTime(created.expires_at)}.
                        It won't be shown again.
                    </p>
                    <div className="flex items-center gap-2">
                        <code className="flex-1 font-mono text-lg bg-base-200 rounded px-3 py-2 select-all">{created.code}</code>
                        <button className="btn btn-sm" onClick={() => copy(created.code, "code")}>
                            {copied === "code" ? "Copied" : "Copy"}
                        </button>
                    </div>
                    <div className="flex items-center gap-2">
                        <code className="flex-1 text-xs bg-base-200 rounded px-3 py-2 break-all select-all">{link}</code>
                        <button className="btn btn-sm" onClick={() => copy(link, "link")}>
                            {copied === "link" ? "Copied" : "Copy link"}
                        </button>
                    </div>
                    <div className="flex justify-end">
                        <button className="btn btn-primary" onClick={close}>Done</button>
                    </div>
                </>}
            </div>
        </dialog>
    );
});
