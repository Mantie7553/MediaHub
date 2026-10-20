import PageHeader from "../../components/layout/PageHeader"
import { useRef, forwardRef, useState } from "react";
import { Check, Download, Inbox, X } from "lucide-react";
import useJobs from "../../hooks/useJobs";
import { useAdminRequests } from "../../hooks"
import { approveRequest, rejectRequest, setJobsArchived, setRequestsArchived } from "../../services/requestsService";
import Format from "../../utils/format";
import { statusBadge, statusColor } from "../../utils/status";
import { toast } from "../../stores/useToastStore";
import StatusPill from "../../components/StatusPill";
import RecordsPanel from "./RecordsPanel";

// pending requests first, the rest newest first
const byPendingThenNewest = (a, b) =>
    (b.status === "pending") - (a.status === "pending") || new Date(b.requested_at) - new Date(a.requested_at);

const requestStatuses = ["pending", "approved", "rejected", "queued", "downloading", "complete", "failed"];
const jobStatuses = ["queued", "downloading", "complete", "failed"];

const requestTitle = item => item.media_title || item.title_override || "Unknown";

/**
 * Downloads and requests page layout
 * @returns
 */
export default function DownloadsPage() {
    const [requestsArchived, setRequestsArchivedView] = useState(false);
    const [jobsArchived, setJobsArchivedView] = useState(false);
    const { requests, loading: rLoading, error: rError, refetch: refetchRequests } = useAdminRequests(requestsArchived)
    const { jobs, loading: jLoading, error: jError, pollError: jPollError, refetch: refetchJobs } = useJobs(jobsArchived);
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

    /**
     * Archive or restore records, with an Undo on the toast
     * @param {function} save setRequestsArchived or setJobsArchived
     * @param {function} refetch reloads the list in the background
     * @param {string} noun what one record is called
     */
    function archiver(save, refetch, noun) {
        return function change(ids, archived, undoing = false) {
            return save(ids, archived)
                .then(() => {
                    refetch(false)
                    if (undoing) return
                    const what = ids.length === 1 ? `${noun[0].toUpperCase()}${noun.slice(1)}` : `${ids.length} ${noun}s`
                    toast.success(`${what} ${archived ? "archived" : "restored"}`,
                        { label: "Undo", onClick: () => change(ids, !archived, true) })
                })
                .catch(err => toast.error(err, `Could not ${archived ? "archive" : "restore"} the ${noun}s`))
        }
    }
    const archiveRequests = archiver(setRequestsArchived, refetchRequests, "request");
    const archiveJobs = archiver(setJobsArchived, refetchJobs, "job");

    const pending = requests.filter(r => r.status === "pending").length;
    const sortedRequests = [...requests].sort(byPendingThenNewest);

    return <div>
        <PageHeader title="Downloads" subtitle="Approve requests and keep an eye on what's downloading."/>
        <div className="grid gap-6 xl:grid-cols-2 items-start">
            <RecordsPanel
                title="Requests"
                icon={Inbox}
                noun="request"
                meta={!rLoading && !rError && !requestsArchived && pending > 0 ? `${pending} pending` : null}
                items={sortedRequests}
                loading={rLoading}
                error={rError}
                onRetry={refetchRequests}
                archived={requestsArchived}
                onArchivedChange={setRequestsArchivedView}
                statuses={requestStatuses}
                titleOf={requestTitle}
                onArchive={archiveRequests}
                empty={{ icon: Inbox, title: "No requests yet", hint: "Requests people make from Discover show up here for approval." }}
                    columns={[
                        { key: "media_title", header: "Title", primary: true, render: (_, item) => <div className="min-w-0">
                            <p className="font-medium truncate max-w-56">{requestTitle(item)}</p>
                            {item.requested_by && <p className="text-xs text-base-content/45">by {item.requested_by}</p>}
                        </div> },
                        { key: "status", header: "Status", render: (val) => <StatusPill label={val} tone={statusBadge(val).replace("badge-", "")}/> },
                        { key: "requested_at", header: "Requested", render: (val) => <span className="text-base-content/60 whitespace-nowrap" title={Format.dateTime(val)}>{Format.shortDateTime(val)}</span> },
                        { key: "actions", header: "", render: (_, item) => item.status === "pending" && (
                            <div className="inline-flex gap-1.5">
                                {/* in the table Reject is just an X, the cards have room for the word */}
                                <button
                                    className="btn btn-sm btn-ghost text-error hover:bg-error/10 @2xl:btn-square"
                                    disabled={busyId === item.id}
                                    onClick={() => { setSelectedRequest(item); modalRef.current.showModal() }}
                                    aria-label="Reject"
                                >
                                    <X size={15}/> <span className="@2xl:hidden">Reject</span>
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
            />
            <RecordsPanel
                title="Jobs"
                icon={Download}
                noun="job"
                meta={!jLoading && !jError && !jobsArchived && jobs.length > 0 ? jobs.length : null}
                items={jobs}
                loading={jLoading}
                error={jError}
                pollError={jPollError}
                onRetry={refetchJobs}
                archived={jobsArchived}
                onArchivedChange={setJobsArchivedView}
                statuses={jobStatuses}
                titleOf={job => job.media_title}
                onArchive={archiveJobs}
                rowKey={job => job.id ?? `${job.media_title}-${job.handler}`}
                empty={{ icon: Download, title: "No downloads running", hint: "Approved requests appear here while they download." }}
                    columns={[
                        { key: "media_title", header: "Title", primary: true, render: (val, item) => <div className="min-w-0">
                            <p className="font-medium truncate max-w-56">{val}</p>
                            {item.status === "failed" && item.error_message && <p className="text-xs text-error/80 truncate max-w-56" title={item.error_message}>{item.error_message}</p>}
                        </div> },
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
            />
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
