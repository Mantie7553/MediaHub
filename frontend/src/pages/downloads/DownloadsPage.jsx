import { useRef, forwardRef, useState } from "react";
import useJobs from "../../hooks/useJobs";
import useRequests from "../../hooks/useRequests"
import { approveRequest, rejectRequest } from "../../services/requestsService";
import Format from "../../utils/format";
import { statusBadge, statusColor } from "../../utils/status";

/**
 * Page where downloading content is handled
 */
export default function DownloadsPage() {

    const { requests, loading: rLoading, error: rError, refetch: refetchRequests } = useRequests()
    const { jobs, loading: jLoading, error: jError } = useJobs();
    const [selectedRequest, setSelectedRequest] = useState(null);
    const modalRef = useRef(null);

    function handleApprove(id) {
        approveRequest(id).then(() => refetchRequests())
    }

    function handleReject(id, notes) {
        rejectRequest(id, notes).then(() => refetchRequests())
    }

    if (rLoading || jLoading) return <div className="flex justify-center p-10"><span className="loading loading-spinner loading-lg"></span></div>

    if (rError || jError) return <div className="alert alert-error">Failed to load data</div>

    return<div className="flex gap-10 justify-center">
        <section className="card">
            <h1>Requests</h1>
            <Table 
            columns={[
            { key: "media_title", header: "Title", render: (val, item) => val || item.title_override || "Unknown" },
            { key: "status", header: "Status", render: (val) => <span className={`badge ${statusBadge(val)}`}>{val}</span> },
            { key: "requested_at", header: "Requested At", render: (val) => Format.dateTime(val) },
            { key: "actions", header: "Actions", render: (_, item) => (
                <div className="flex gap-2">
                    <button 
                    className="btn btn-xs btn-success"
                    disabled={item.status !== "pending"}
                    onClick={() => handleApprove(item.id)}
                    >
                        Approve
                    </button>
                    <button 
                    className="btn btn-xs btn-error"
                    disabled={item.status !== "pending"}
                    onClick={() => { setSelectedRequest(item); modalRef.current.showModal() }}
                    >
                        Reject
                    </button>
                </div>
                )},
            ]}
            items={requests}
            />
        </section>
        <section className="card">
            <h1>Jobs</h1>
            <Table 
            columns={[
                { key: "media_title", header: "Title" },
                { key: "handler", header: "Handler"},
                { key: "progress_pct", header: "Progress", render: (val) => (
                    <progress className="progress progress-primary min-w-30" value={val} max="100"></progress>
                )},
                { key: "status", header: "Status", render: (val) => (
                    <div className="flex gap-2 items-center">
                        <div className="inline-grid *:[grid-area:1/1]">
                        <div className={`status ${statusColor(val)} animate-ping`}></div>
                        <div className={`status ${statusColor((val))}`}></div>
                        </div>
                    </div>
                )},
            ]} 
            items={jobs}
            />
        </section>
        <Modal ref={modalRef} onConfirm={(notes) => handleReject(selectedRequest.id, notes)} />
    </div>
}

/**
 *  Builds a table with appropraite column headings and fields
 * @param columns the heading information for the table
 * @param items the data we want to display
 * @returns a table representing some data
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
    )
}

const Modal = forwardRef(function Modal({onConfirm}, ref) {
const notesRef = useRef(null);

    return <dialog id="my_modal_3" className="modal" ref={ref}>
    <div className="modal-box">
        <form method="dialog">
        <button className="btn btn-sm btn-circle btn-ghost absolute right-2 top-2">✕</button>
        </form>
        <h3 className="font-bold text-lg">Reject Download Request</h3>
        <p className="py-4">Confirm rejection of the Download Request</p>
        <label className="flex flex-col gap-2 p-2">
            Notes:
            <textarea ref={notesRef} className="textarea w-full h-32"/>
        </label>
        <div className="flex gap-2 justify-center">
            <button
            className="btn btn-error"
            onClick={() => {
                onConfirm(notesRef.current.value);
                ref.current.close();
            }}
            >
                Confirm
            </button>
            <button
            className="btn btn-info"
            onClick={() => ref.current.close()}
            >
                Cancel
            </button>
        </div>
    </div>
  </dialog>
})
