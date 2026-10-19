import PageHeader from "../../components/layout/PageHeader"
import { useRef, forwardRef, useState } from "react";
import { Check, Download, Inbox, WifiOff, X } from "lucide-react";
import useJobs from "../../hooks/useJobs";
import { useAdminRequests } from "../../hooks"
import { approveRequest, rejectRequest } from "../../services/requestsService";
import Format from "../../utils/format";
import { statusBadge, statusColor } from "../../utils/status";
import { errorMessage } from "../../utils/errors";
import { toast } from "../../stores/useToastStore";
import ErrorState from "../../components/states/ErrorState";
import EmptyState from "../../components/states/EmptyState";
import { TableSkeleton } from "../../components/states/Skeletons";
import DataTable, { Panel } from "../../components/DataTable";
import StatusPill from "../../components/StatusPill";

// pending requests first, the rest newest first
const byPendingThenNewest = (a, b) =>
    (b.status === "pending") - (a.status === "pending") || new Date(b.requested_at) - new Date(a.requested_at);

/**
 * Downloads and requests page layout
 * @returns
 */
export default function DownloadsPage() {

    const { requests, loading: rLoading, error: rError, refetch: refetchRequests } = useAdminRequests()
    const { jobs, loading: jLoading, error: jError, pollError: jPollError, refetch: refetchJobs } = useJobs();
    const [selectedRequest, setSelectedRequest] = useState(null);
    const [busyId, setBusyId] = useState(null);
    const modalRef = useRef(null);

    function handleApprove(id) {
        setBusyId(id)
        approveRequest(id)
            .then(() => { toast.success("Request approved, the download is starting"); refetchRequests(false) })
            .catch(err => toast.error(err, "Could not approve the request"))
            .finally(() => setBusyId(null))
    }

    function handleReject(id, notes) {
        setBusyId(id)
        rejectRequest(id, notes)
            .then(() => { toast.success("Request rejected"); refetchRequests(false) })
            .catch(err => toast.error(err, "Could not reject the request"))
            .finally(() => setBusyId(null))
    }

    const pending = requests.filter(r => r.status === "pending").length;
    const sortedRequests = [...requests].sort(byPendingThenNewest);

    return <div>
        <PageHeader title="Downloads" subtitle="Approve requests and keep an eye on what's downloading."/>
        <div className="grid gap-6 xl:grid-cols-2 items-start">
            <Panel
                title="Requests"
                icon={Inbox}
                meta={!rLoading && !rError && (pending > 0 ? `${pending} pending` : requests.length || null)}
            >
                {rLoading ? <TableSkeleton columns={4}/> :
                rError ? <ErrorState compact error={rError} onRetry={refetchRequests}/> :
                requests.length === 0 ? <EmptyState compact title="No requests yet" hint="Requests people make from Discover show up here for approval."/> :
                <DataTable
                    items={sortedRequests}
                    columns={[
                        { key: "media_title", header: "Title", primary: true, render: (val, item) => <div className="min-w-0">
                            <p className="font-medium truncate max-w-xs">{val || item.title_override || "Unknown"}</p>
                            {item.requested_by && <p className="text-xs text-base-content/45">by {item.requested_by}</p>}
                        </div> },
                        { key: "status", header: "Status", render: (val) => <StatusPill label={val} tone={statusBadge(val).replace("badge-", "")}/> },
                        { key: "requested_at", header: "Requested", render: (val) => <span className="text-base-content/60 whitespace-nowrap">{Format.dateTime(val)}</span> },
                        { key: "actions", header: "", render: (_, item) => item.status === "pending" && (
                            <div className="inline-flex gap-1.5">
                                <button
                                    className="btn btn-sm btn-ghost text-error hover:bg-error/10"
                                    disabled={busyId === item.id}
                                    onClick={() => { setSelectedRequest(item); modalRef.current.showModal() }}
                                >
                                    <X size={15}/> Reject
                                </button>
                                <button
                                    className="btn btn-sm btn-primary"
                                    disabled={busyId === item.id}
                                    onClick={() => handleApprove(item.id)}
                                >
                                    {busyId === item.id ? <span className="loading loading-spinner loading-xs"/> : <Check size={15}/>} Approve
                                </button>
                            </div>
                        )},
                    ]}
                />}
            </Panel>
            <Panel
                title="Jobs"
                icon={Download}
                meta={!jLoading && !jError && jobs.length > 0 ? jobs.length : null}
            >
                {jPollError && <p className="flex items-center gap-2 text-xs text-warning mb-3" role="status">
                    <WifiOff size={14}/> Lost connection, retrying: {errorMessage(jPollError)}
                </p>}
                {jLoading ? <TableSkeleton columns={4}/> :
                jError ? <ErrorState compact error={jError} onRetry={refetchJobs}/> :
                jobs.length === 0 ? <EmptyState compact icon={Download} title="No downloads running" hint="Approved requests appear here while they download."/> :
                <DataTable
                    items={jobs}
                    rowKey={job => job.id ?? `${job.media_title}-${job.handler}`}
                    columns={[
                        { key: "media_title", header: "Title", primary: true, render: (val) => <p className="font-medium truncate max-w-xs">{val}</p> },
                        { key: "handler", header: "Handler", render: (val) => <span className="text-base-content/60 capitalize">{val}</span> },
                        { key: "progress_pct", header: "Progress", render: (val) => (
                            <div className="flex items-center justify-end md:justify-start gap-2.5">
                                <div className="h-1.5 w-28 rounded-full bg-white/10 overflow-hidden">
                                    <div className="h-full rounded-full bg-primary transition-[width] duration-500" style={{ width: `${Math.round(val ?? 0)}%` }}></div>
                                </div>
                                <span className="w-9 text-xs tabular-nums text-base-content/60">{Math.round(val ?? 0)}%</span>
                            </div>
                        )},
                        { key: "status", header: "Status", render: (val) => (
                            <StatusPill label={val} tone={statusColor(val).replace("status-", "")} pulse={val === "downloading" || val === "queued"}/>
                        )},
                    ]}
                />}
            </Panel>
        </div>
        <Modal ref={modalRef} request={selectedRequest} onConfirm={(notes) => handleReject(selectedRequest.id, notes)} />
    </div>
}

/**
 * A confirmation modal for rejecting a request
 */
const Modal = forwardRef(function Modal({request, onConfirm}, ref) {
const notesRef = useRef(null);

    return <dialog className="modal modal-bottom sm:modal-middle" ref={ref} onClose={() => { if (notesRef.current) notesRef.current.value = "" }}>
    <div className="modal-box flex flex-col gap-4">
        <form method="dialog">
            <button className="btn btn-sm btn-circle btn-ghost absolute right-3 top-3" aria-label="Close">
                <X size={16}/>
            </button>
        </form>
        <div>
            <h3 className="text-xl font-semibold">Reject request</h3>
            {request && <p className="text-sm text-base-content/60 mt-1">
                {request.media_title || request.title_override || "This title"}
                {request.requested_by ? `, requested by ${request.requested_by}` : ""}
            </p>}
        </div>
        <label className="flex flex-col gap-1.5">
            <span className="text-xs font-medium uppercase tracking-wider text-base-content/55">Note for the requester (optional)</span>
            <textarea ref={notesRef} className="textarea w-full h-28 bg-base-100/60 border-white/10 text-base sm:text-sm" placeholder="Why it was rejected"/>
        </label>
        <div className="flex gap-2 justify-end">
            <button className="btn btn-ghost" onClick={() => ref.current.close()}>
                Cancel
            </button>
            <button
                className="btn btn-error"
                onClick={() => {
                    onConfirm(notesRef.current.value);
                    ref.current.close();
                }}
            >
                Reject request
            </button>
        </div>
    </div>
  </dialog>
})
