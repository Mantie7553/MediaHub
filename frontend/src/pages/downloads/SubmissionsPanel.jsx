import { useRef, useState } from "react";
import { Link } from "react-router-dom";
import { Check, Sparkles, X } from "lucide-react";
import DataTable, { Panel } from "../../components/DataTable";
import { CollectionCover } from "../../components/cards";
import NoteModal from "../../components/modals/NoteModal";
import StatusPill from "../../components/StatusPill";
import ErrorState from "../../components/states/ErrorState";
import useApi from "../../hooks/useApi";
import { approveSubmission, rejectSubmission } from "../../services/collectionsService";
import { toast } from "../../stores/useToastStore";
import Format from "../../utils/format";

const pathOf = (kind, id) => `/${kind === "collection" ? "collections" : "playlists"}/${id}`;

/**
 * Playlists and collections people asked to have featured. Approving freezes a copy of one as it
 * is now for everyone's Featured row (replacing the copy made from it before, if there is one);
 * turning one down sends the reason back to its owner. Shows only while something is waiting
 */
export default function SubmissionsPanel() {
    const { data: submissions, loading, error, reload } = useApi("/admin/collection-submissions", { initial: [] });
    const [busyId, setBusyId] = useState(null);
    const [rejecting, setRejecting] = useState(null);
    const rejectRef = useRef(null);

    function approve(item) {
        setBusyId(item.id);
        approveSubmission(item.id)
            .then(() => { toast.success(item.updates_id ? `Updated the featured ${item.collection.name}` : `Featured ${item.collection.name}`); reload(true) })
            .catch(err => toast.error(err, "Could not feature it"))
            .finally(() => setBusyId(null));
    }

    function reject(note) {
        return rejectSubmission(rejecting.id, note)
            .then(() => { toast.success("Turned it down"); reload(true) });
    }

    if (loading || (!error && submissions.length === 0)) return null;

    return <Panel title="Featured submissions" icon={Sparkles} meta={error ? null : `${submissions.length} waiting`}
        description="Approving puts a copy of it, as it is now, in everyone's Featured row. Open one to look through it first.">
        {error ? <ErrorState compact error={error} onRetry={reload}/> : <DataTable
            items={submissions}
            columns={[
                { key: "collection", header: "Playlist or collection", primary: true, render: (c, item) => (
                    <div className="flex items-center gap-3 min-w-0">
                        <CollectionCover covers={c.covers} kind={c.kind} className="size-11 shrink-0 rounded-md overflow-hidden"/>
                        <div className="min-w-0">
                            <Link to={pathOf(c.kind, c.id)} className="block font-medium truncate hover:text-primary transition-colors">{c.name}</Link>
                            <p className="text-xs text-base-content/45 truncate">
                                {c.kind === "collection" ? "Collection" : "Playlist"} · {c.item_count} {c.item_count === 1 ? "item" : "items"} · by {item.submitted_by}
                            </p>
                            {item.note && <p className="text-xs text-base-content/70 mt-0.5 break-words">“{item.note}”</p>}
                        </div>
                    </div>
                )},
                { key: "updates_name", header: "Featured", render: (val, item) => item.updates_id
                    ? <Link to={pathOf(item.collection.kind, item.updates_id)}><StatusPill label="Updates featured" tone="info"/></Link>
                    : <StatusPill label="New" tone="primary"/> },
                { key: "submitted_at", header: "Sent", render: val => (
                    <span className="text-base-content/60 whitespace-nowrap" title={Format.dateTime(val)}>{Format.shortDateTime(val)}</span>
                )},
                { key: "actions", header: "", render: (_, item) => (
                    <div className="inline-flex gap-1.5">
                        <button className="btn btn-sm btn-ghost text-error hover:bg-error/10 @2xl:btn-square" disabled={busyId === item.id}
                            onClick={() => { setRejecting(item); rejectRef.current.showModal() }} aria-label="Turn down">
                            <X size={15}/> <span className="@2xl:hidden">Turn down</span>
                        </button>
                        <button className="btn btn-sm btn-primary" disabled={busyId === item.id} onClick={() => approve(item)}>
                            {busyId === item.id ? <span className="loading loading-spinner loading-xs"/> : <Check size={15}/>} Approve
                        </button>
                    </div>
                )},
            ]}
        />}
        <NoteModal dialogRef={rejectRef} tone="error"
            title={rejecting ? `Turn down “${rejecting.collection.name}”?` : "Turn it down?"}
            intro={rejecting ? `${rejecting.submitted_by} sees your reason on it and can change it and send it again.` : null}
            label="Reason (optional)"
            confirmLabel="Turn down"
            onSave={reject}/>
    </Panel>;
}
