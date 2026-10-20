import { useState } from "react"
import { Archive, ArchiveRestore, Search, WifiOff } from "lucide-react"
import DataTable, { Panel } from "../../components/DataTable"
import SegmentedTabs from "../../components/SegmentedTabs"
import TextField from "../../components/TextField"
import ErrorState from "../../components/states/ErrorState"
import EmptyState from "../../components/states/EmptyState"
import { TableSkeleton } from "../../components/states/Skeletons"
import { errorMessage } from "../../utils/errors"
import { filterRecords, statusCounts } from "../../utils/records"

const views = [
    { key: "active", label: "Active" },
    { key: "archived", label: "Archived", icon: Archive },
]

/**
 * A Downloads panel of requests or jobs that can be filtered by status and title, and archived:
 * hidden from the list without being deleted, one row at a time or every row shown at once.
 * The Archived view lists them again so they can be restored
 * @param {string} title the heading
 * @param {any} icon a lucide icon for the heading
 * @param {any} meta a note beside the title, like a count
 * @param {object[]} items the records in the current view, each with id and status
 * @param {boolean} loading shows a placeholder table
 * @param {any} error the load error, shown with a retry
 * @param {any} pollError a background refresh failed while older records are shown
 * @param {function} onRetry loads again after an error
 * @param {boolean} archived whether the Archived view is shown
 * @param {function} onArchivedChange called with true or false to switch views
 * @param {string[]} statuses the statuses in the order their tabs appear
 * @param {function} titleOf reads a record's title, for the title filter
 * @param {object[]} columns DataTable columns; an "actions" column gets the archive button added
 * @param {function} onArchive called with (ids, archived) to archive or restore records
 * @param {string} noun what one record is called, like "request"
 * @param {object} empty { title, hint, icon } when the Active view has nothing at all
 * @param {function} rowKey picks a record's key
 */
export default function RecordsPanel({ title, icon, meta, items, loading, error, pollError, onRetry,
    archived, onArchivedChange, statuses, titleOf, columns, onArchive, noun, empty, rowKey }) {
    const [status, setStatus] = useState("all")
    const [query, setQuery] = useState("")

    const counts = statusCounts(items, statuses)
    // a status whose last record was archived falls back to All instead of an empty tab
    const activeStatus = counts.some(c => c.key === status) ? status : "all"
    const shown = filterRecords(items, activeStatus, query, titleOf)
    const filtered = activeStatus !== "all" || query.trim() !== ""
    const plural = n => `${n} ${noun}${n === 1 ? "" : "s"}`

    const tabs = [
        { key: "all", label: "All", count: items.length },
        ...counts.map(({ key, count }) => ({ key, label: key[0].toUpperCase() + key.slice(1), count })),
    ]

    const actionsColumn = columns.find(col => col.key === "actions")
    const archiveButton = item => <button
        className="btn btn-sm btn-ghost btn-square text-base-content/50 hover:text-base-content"
        onClick={() => onArchive([item.id], !archived)}
        aria-label={archived ? "Restore" : "Archive"}
        title={archived ? "Restore" : "Archive"}
    >
        {archived ? <ArchiveRestore size={15}/> : <Archive size={15}/>}
    </button>
    const tableColumns = [
        ...columns.filter(col => col !== actionsColumn),
        { key: "actions", header: "", render: (val, item) => <div className="inline-flex items-center gap-1.5">
            {!archived && actionsColumn?.render(val, item)}
            {archiveButton(item)}
        </div> },
    ]

    // the whole Active list can't be archived in one go: pick a status or filter first, so
    // pending requests and running jobs aren't swept away by accident
    const bulk = shown.length > 0 && (archived || filtered) && <button
        className="btn btn-sm btn-ghost border-white/10 bg-white/5 hover:bg-white/10 gap-1.5"
        onClick={() => onArchive(shown.map(item => item.id), !archived)}
    >
        {archived ? <ArchiveRestore size={15}/> : <Archive size={15}/>}
        {archived ? "Restore" : "Archive"} {shown.length === items.length && !filtered ? "all" : plural(shown.length)}
    </button>

    function body() {
        if (loading) return <TableSkeleton columns={4}/>
        if (error) return <ErrorState compact error={error} onRetry={onRetry}/>
        if (items.length === 0) return archived
            ? <EmptyState compact icon={Archive} title={`No archived ${noun}s`} hint={`Archive a ${noun} to tidy the Active list without deleting it.`}/>
            : <EmptyState compact icon={empty.icon} title={empty.title} hint={empty.hint}/>
        return <>
            <div className="flex flex-col gap-3 mb-4">
                <SegmentedTabs options={tabs} value={activeStatus} onChange={setStatus} label={`Filter ${noun}s by status`} bleed={false}/>
                <div className="flex flex-wrap items-center gap-2">
                    <div className="flex-1 min-w-48">
                        <TextField icon={Search} type="search" value={query} onChange={e => setQuery(e.target.value)}
                            placeholder="Filter by title" aria-label={`Filter ${noun}s by title`}/>
                    </div>
                    {bulk}
                </div>
            </div>
            {shown.length === 0
                ? <EmptyState compact icon={Search} title="Nothing matches" hint="Try another status or title."/>
                : <DataTable items={shown} rowKey={rowKey} columns={tableColumns}/>}
        </>
    }

    return <Panel
        title={title}
        icon={icon}
        meta={meta}
        actions={<SegmentedTabs options={views} value={archived ? "archived" : "active"}
            onChange={key => onArchivedChange(key === "archived")} label={`${title} view`} bleed={false}/>}
    >
        {pollError && <p className="flex items-center gap-2 text-xs text-warning mb-3" role="status">
            <WifiOff size={14}/> Lost connection, retrying: {errorMessage(pollError)}
        </p>}
        {body()}
    </Panel>
}
