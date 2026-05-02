import useJobs from "../../hooks/useJobs";
import useRequests from "../../hooks/useRequests"
import { approveRequest, rejectRequest } from "../../services/requestsService";
import Format from "../../utils/format";

/**
 * Page where downloading content is handled
 */
export default function DownloadsPage() {

    const { requests, loading: rLoading, error: rError, refetch: refetchRequests } = useRequests()
    const { jobs, loading: jLoading, error: jError } = useJobs();

    function handleApprove(id) {
        approveRequest(id).then(() => refetchRequests())
    }

    function handleReject(id) {
        rejectRequest(id).then(() => refetchRequests())
    }

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
                    className={`btn btn-xs ${item.status === "pending" ? "btn-success" : "btn-neutral"}`}
                    onClick={() => handleApprove(item.id)}
                    >
                        Approve
                    </button>
                    <button 
                    className={`btn btn-xs ${item.status === "pending" ? "btn-error" : "btn-neutral"}`}
                    onClick={() => handleReject(item.id)}
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
                { key: "title", header: "Title" },
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

/**
 * Helper function for translating status into color
 * @param status the status we are matching the color to
 * @returns the class representation of the color we want displayed
 */
function statusBadge(status) {
    switch(status) {
        case "approved": return "badge-success"
        case "pending": return "badge-warning"
        case "rejected": return "badge-error"
        default: return "badge-neutral"
    }
}

/**
 * Helper function for translating status into color
 * @param status the status we are matching the color to
 * @returns the class representation of the color we want displayed
 */
function statusColor(status) {
    switch(status) {
        case "complete": return "status-success"
        case "downloading": return "status-info"
        case "queued": return "status-warning"
        case "failed": return "status-error"
        default: return "status-neutral"
    }
}