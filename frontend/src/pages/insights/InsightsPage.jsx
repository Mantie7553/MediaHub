import { AlertTriangle, BookOpen, Clapperboard, Download, HardDrive, RefreshCw, Users } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router-dom";
import PageHeader from "../../components/layout/PageHeader";
import DataTable, { Panel } from "../../components/DataTable";
import StatTile from "../../components/StatTile";
import StatusPill from "../../components/StatusPill";
import EmptyState from "../../components/states/EmptyState";
import ErrorState from "../../components/states/ErrorState";
import { TableSkeleton } from "../../components/states/Skeletons";
import useApi from "../../hooks/useApi";
import api from "../../services/api";
import { toast } from "../../stores/useToastStore";
import Format from "../../utils/format";

const detailsPaths = { anime: "/anime/", movie: "/movies/", manga: "/manga/", light_novel: "/light-novels/" };
const typeLabels = { anime: "Anime", movie: "Movie", manga: "Manga", light_novel: "Light novel" };
const folderLabels = { anime: "Anime", movie: "Movies", manga: "Manga", light_novel: "Light novels", music: "Music" };
const jobTones = { complete: "success", failed: "error", downloading: "info", queued: "warning" };

/**
 * Admin page showing how the server is doing: disk use, what people watch and read, who has been
 * around and how downloads are going
 */
export default function InsightsPage() {
    const { data, loading, error, reload } = useApi("/admin/insights");
    const [rescanning, setRescanning] = useState(false);

    function rescan() {
        setRescanning(true);
        api.get("/admin/insights?refresh=1")
            .then(() => reload(true))
            .catch(err => toast.error(err, "Could not scan the library"))
            .finally(() => setRescanning(false));
    }

    const header = <PageHeader title="Insights" subtitle="Disk use, what people watch and read, and how downloads are going." />;

    if (loading) return <div className="flex flex-col gap-6">
        {header}
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
            {Array.from({ length: 4 }, (_, i) => <div key={i} className="skeleton h-24"></div>)}
        </div>
        <div className="surface p-4 sm:p-6"><TableSkeleton columns={3} rows={5}/></div>
    </div>;
    if (error || !data) return <div className="flex flex-col gap-6">{header}<ErrorState error={error} onRetry={reload}/></div>;

    const { storage, cache, users, jobs } = data;
    const failed = jobs.counts.failed ?? 0;
    const lowDisk = storage.disk_total && storage.disk_free / storage.disk_total < 0.1;

    return (
        <div className="flex flex-col gap-6">
            {header}

            <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
                <StatTile icon={HardDrive} label="Disk free"
                    value={Format.bytes(storage.disk_free)}
                    hint={storage.disk_total != null ? `of ${Format.bytes(storage.disk_total)}` : "Not available"}
                    tone={lowDisk ? "warning" : undefined}/>
                <StatTile icon={HardDrive} label="File cache"
                    value={cache.enabled ? Format.bytes(cache.used) : "Off"}
                    hint={cache.enabled ? `of ${Format.bytes(cache.max)}` : "CACHE_MAX_GB is 0"}/>
                <StatTile icon={Users} label="Active this week" value={users.active_7d}
                    hint={`${users.active_30d} this month · ${users.total} total`}/>
                <StatTile icon={AlertTriangle} label="Failed downloads" value={failed}
                    hint={`30 days · ${jobs.pending_requests} awaiting approval`}
                    tone={failed > 0 ? "error" : undefined}/>
            </div>

            <StoragePanel storage={storage} rescanning={rescanning} onRescan={rescan}/>

            <div className="grid gap-6 xl:grid-cols-2">
                <TopPanel title="Most watched" icon={Clapperboard} items={data.top_watched} itemsHeader="Episodes" showHours
                    empty="Nothing watched in the last 30 days"/>
                <TopPanel title="Most read" icon={BookOpen} items={data.top_read} itemsHeader="Chapters, volumes"
                    empty="Nothing read in the last 30 days"/>
            </div>

            <Panel title="Users" icon={Users} meta={users.total} description="Last seen is the latest sign in or saved progress.">
                <DataTable
                    items={users.list}
                    columns={[
                        { key: "username", header: "User", primary: true, render: val => <span className="font-medium truncate">{val}</span> },
                        { key: "role", header: "Role", render: val => <StatusPill label={Format.cleanString(val)} tone={val === "admin" ? "primary" : "secondary"}/> },
                        { key: "last_seen", header: "Last seen", render: val => (
                            <span className="text-base-content/60 whitespace-nowrap" title={val ? Format.dateTime(val) : undefined}>{Format.relativeTime(val)}</span>
                        )},
                        { key: "recent_titles", header: "Titles, 30 days", render: val => <span className="tabular-nums">{val}</span> },
                    ]}
                />
            </Panel>

            <Panel title="Downloads" icon={Download} description="Jobs started in the last 30 days, and failures still on the Downloads page."
                actions={<Link to="/downloads" className="btn btn-sm btn-ghost">Open Downloads</Link>}>
                <div className="flex flex-wrap gap-2 mb-5">
                    {["complete", "downloading", "queued", "failed"].map(status => (
                        <StatusPill key={status} tone={jobTones[status]} label={`${Format.cleanString(status)} · ${jobs.counts[status] ?? 0}`}/>
                    ))}
                </div>
                {jobs.recent_failures.length === 0
                    ? <EmptyState compact title="No failed downloads" hint="Failures show up here until they're archived."/>
                    : <DataTable
                        items={jobs.recent_failures}
                        columns={[
                            { key: "title", header: "Title", primary: true, render: val => <span className="font-medium">{val}</span> },
                            { key: "handler", header: "Via", render: val => <span className="text-base-content/60">{Format.cleanString(val)}</span> },
                            { key: "error", header: "Error", className: "max-w-md", render: val => (
                                <span className="text-error/90 text-xs break-words">{val || "No message"}</span>
                            )},
                            { key: "failed_at", header: "When", render: val => (
                                <span className="text-base-content/60 whitespace-nowrap">{val ? Format.shortDateTime(val) : "—"}</span>
                            )},
                        ]}
                    />}
            </Panel>
        </div>
    );
}

/**
 * How much each library folder holds, as bars against the largest one
 */
function StoragePanel({ storage, rescanning, onRescan }) {
    const library = storage.folders.reduce((sum, f) => sum + f.bytes, 0);
    const largest = Math.max(1, ...storage.folders.map(f => f.bytes));
    const used = storage.disk_total != null ? storage.disk_total - storage.disk_free : null;

    return <Panel title="Storage" icon={HardDrive} meta={Format.bytes(library)}
        description={`Scanned ${Format.relativeTime(storage.scanned_at)}, kept for an hour.`}
        actions={<button className="btn btn-sm btn-ghost" onClick={onRescan} disabled={rescanning}>
            <RefreshCw size={15} className={rescanning ? "animate-spin" : ""}/> Rescan
        </button>}>
        {used != null && <div className="mb-6">
            <div className="flex justify-between gap-3 text-sm mb-1.5">
                <span className="text-base-content/70">Disk</span>
                <span className="tabular-nums text-base-content/60 text-right">{Format.bytes(used)} used of {Format.bytes(storage.disk_total)}</span>
            </div>
            <Bar fraction={storage.disk_total ? used / storage.disk_total : 0}/>
        </div>}
        <ul className="flex flex-col gap-4">
            {storage.folders.map(f => (
                <li key={f.type}>
                    <div className="flex justify-between gap-3 text-sm mb-1.5">
                        <span className="min-w-0 truncate">{folderLabels[f.type] ?? f.name} <span className="text-base-content/40">· {f.files.toLocaleString()} files</span></span>
                        <span className="tabular-nums text-base-content/70 shrink-0">{Format.bytes(f.bytes)}</span>
                    </div>
                    <Bar fraction={f.bytes / largest}/>
                </li>
            ))}
        </ul>
        <p className="mt-5 text-sm text-base-content/55">
            Transcodes and unfinished uploads: <span className="tabular-nums text-base-content/80">{Format.bytes(storage.temp.bytes)}</span> in {storage.temp.files.toLocaleString()} files, cleared after 48 hours.
        </p>
    </Panel>;
}

function Bar({ fraction }) {
    const pct = Math.min(100, Math.max(0, fraction * 100));
    return <div className="h-2 rounded-full bg-white/8 overflow-hidden">
        <div className="h-full rounded-full bg-linear-to-r from-secondary to-primary" style={{ width: `${pct}%` }}></div>
    </div>;
}

/**
 * The titles the most people watched or read in the last 30 days
 */
function TopPanel({ title, icon, items, itemsHeader, showHours = false, empty }) {
    return <Panel title={title} icon={icon} description="Last 30 days, by how many people.">
        {items.length === 0
            ? <EmptyState compact title={empty}/>
            : <DataTable
                items={items}
                columns={[
                    { key: "title", header: "Title", primary: true, render: (val, item) => (
                        <Link to={`${detailsPaths[item.type] ?? "/"}${item.id}`} className="font-medium hover:text-primary transition-colors">{val}</Link>
                    )},
                    { key: "type", header: "Type", render: val => <span className="text-base-content/60">{typeLabels[val] ?? val}</span> },
                    { key: "users", header: "People", render: val => <span className="tabular-nums">{val}</span> },
                    { key: "items", header: itemsHeader, render: val => <span className="tabular-nums">{val}</span> },
                    ...(showHours ? [{ key: "hours", header: "Hours", render: val => <span className="tabular-nums">{val.toFixed(1)}</span> }] : []),
                ]}
            />}
    </Panel>;
}
