import { forwardRef, useRef, useState, useEffect } from "react";
import { Check, Copy, StickyNote, Ticket, X } from "lucide-react";
import api from "../../services/api";
import Format from "../../utils/format";
import { errorMessage } from "../../utils/errors";
import { toast } from "../../stores/useToastStore";
import DataTable, { Panel } from "../../components/DataTable";
import StatusPill from "../../components/StatusPill";
import EmptyState from "../../components/states/EmptyState";
import TextField, { SelectField } from "../../components/TextField";

const STATUS_TONE = {
    active: "success",
    used: "secondary",
    expired: "neutral",
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
            .then(res => { setInvites(res.data ?? []); setError(""); })
            .catch(err => setError(errorMessage(err, "Could not load invites")));
    }

    useEffect(() => { fetchInvites(); }, []);

    function handleRevoke(invite) {
        api.delete(`/admin/invites/${invite.id}`)
            .then(() => { toast.success("Invite revoked"); fetchInvites(); })
            .catch(err => toast.error(err, "Could not revoke the invite"));
    }

    const active = invites.filter(i => i.status === "active").length;

    return (
        <Panel
            title="Invites"
            icon={Ticket}
            meta={active > 0 ? `${active} active` : null}
            description="New people need an invite code to sign up."
            actions={<button className="btn btn-sm btn-primary" onClick={() => createModalRef.current.showModal()}>
                <Ticket size={15}/> Create invite
            </button>}
        >
            {error && <p role="alert" className="text-error text-sm mb-3">{error}</p>}
            {invites.length === 0 ? (
                !error && <EmptyState compact icon={Ticket} title="No invites yet" hint="Create one and share the code or link with whoever you're inviting."/>
            ) : (
                <DataTable
                    items={invites}
                    columns={[
                        { key: "code_hint", header: "Code", primary: true, render: (val, invite) => <div className="flex items-center gap-3 min-w-0">
                            <code className="font-mono text-sm rounded-md bg-white/5 ring-1 ring-white/10 px-2 py-0.5">…{val}</code>
                            {invite.note && <span className="text-sm text-base-content/60 truncate md:hidden">{invite.note}</span>}
                        </div> },
                        { key: "note", header: "Note", hideOnMobile: true, className: "max-w-48 truncate text-base-content/70", render: (val) => val ?? "" },
                        { key: "status", header: "Status", render: (val) => <StatusPill label={val} tone={STATUS_TONE[val]}/> },
                        { key: "expires_at", header: "Expires", render: (val) => <span className="text-base-content/60 whitespace-nowrap" title={Format.dateTime(val)}>{Format.shortDateTime(val)}</span> },
                        { key: "used_by", header: "Used by", render: (val) => val ?? <span className="text-base-content/30">—</span> },
                        { key: "created_by", header: "Created by", render: (val) => <span className="text-base-content/60">{val ?? ""}</span> },
                        { key: "actions", header: "", render: (_, invite) => invite.status === "active" && (
                            <button className="btn btn-sm btn-ghost text-error hover:bg-error/10" onClick={() => handleRevoke(invite)}>
                                <X size={15}/> Revoke
                            </button>
                        )},
                    ]}
                />
            )}

            <CreateInviteModal ref={createModalRef} onCreated={fetchInvites} />
        </Panel>
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
            .catch(err => setError(errorMessage(err, "Could not create the invite")))
            .finally(() => setLoading(false));
    }

    function copy(text, what) {
        navigator.clipboard?.writeText(text)
            .then(() => setCopied(what))
            .catch(() => {
                setCopied("");
                toast.error("Couldn't copy, select the text and copy it by hand");
            });
    }

    function close() {
        ref.current.close();
    }

    return (
        <dialog className="modal modal-bottom sm:modal-middle" ref={ref} onClose={reset}>
            <div className="modal-box flex flex-col gap-4">
                <div className="flex items-center gap-3">
                    <span className="grid size-10 place-items-center rounded-full bg-primary/15 text-primary"><Ticket size={18}/></span>
                    <h3 className="text-xl font-semibold">{created ? "Invite created" : "Create invite"}</h3>
                </div>

                {!created ? <>
                    <TextField label="Note (optional)" icon={StickyNote} value={note} onChange={e => setNote(e.target.value)} placeholder="Who is this for?" />
                    <SelectField label="Expires after" value={days} onChange={e => setDays(e.target.value)}>
                        <option value={1}>1 day</option>
                        <option value={7}>7 days</option>
                        <option value={30}>30 days</option>
                    </SelectField>

                    {error && <p role="alert" className="text-error text-sm">{error}</p>}

                    <div className="flex gap-2 justify-end pt-1">
                        <button className="btn btn-ghost" onClick={close}>Cancel</button>
                        <button className="btn btn-primary" onClick={handleCreate} disabled={loading}>
                            {loading && <span className="loading loading-spinner loading-xs"/>} Create
                        </button>
                    </div>
                </> : <>
                    <p className="text-sm text-base-content/70">
                        Share this code or link. It works for one sign up and expires {Format.dateTime(created.expires_at)}.
                        It won't be shown again.
                    </p>
                    <CopyRow label="Code" value={created.code} mono large copied={copied === "code"} onCopy={() => copy(created.code, "code")}/>
                    <CopyRow label="Link" value={link} copied={copied === "link"} onCopy={() => copy(link, "link")}/>
                    <div className="flex justify-end pt-1">
                        <button className="btn btn-primary" onClick={close}>Done</button>
                    </div>
                </>}
            </div>
        </dialog>
    );
});

/**
 * A value to hand out, with a copy button that confirms once it has copied
 */
function CopyRow({ label, value, mono = false, large = false, copied, onCopy }) {
    return <div className="flex flex-col gap-1.5">
        <span className="text-xs font-medium uppercase tracking-wider text-base-content/55">{label}</span>
        <div className="flex items-center gap-2 rounded-field bg-base-100/60 ring-1 ring-white/10 pl-3 pr-1.5 py-1.5">
            <code className={`flex-1 min-w-0 select-all ${mono ? "font-mono" : ""} ${large ? "text-lg tracking-wide" : "text-xs break-all"}`}>{value}</code>
            <button className={`btn btn-sm shrink-0 ${copied ? "btn-success" : "btn-ghost bg-white/5"}`} onClick={onCopy}>
                {copied ? <><Check size={15}/> Copied</> : <><Copy size={15}/> Copy</>}
            </button>
        </div>
    </div>
}
