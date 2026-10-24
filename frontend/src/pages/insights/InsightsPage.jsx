import { Activity, AlertTriangle, BookOpen, Clapperboard, Download, HardDrive, Music, RefreshCw, Users } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router-dom";
import PageHeader from "../../components/layout/PageHeader";
import DataTable, { Panel } from "../../components/DataTable";
import StatTile from "../../components/StatTile";
import ColumnChart from "../../components/charts/ColumnChart";
import LineChart from "../../components/charts/LineChart";
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

            <StoragePanel storage={storage} history={data.storage_history} rescanning={rescanning} onRescan={rescan}/>

            <div className="grid gap-6 xl:grid-cols-2">
                <TopPanel title="Most watched" icon={Clapperboard} items={data.top_watched} itemsHeader="Episodes" showHours
                    empty="Nothing watched in the last 30 days"/>
                <TopPanel title="Most read" icon={BookOpen} items={data.top_read} itemsHeader="Chapters, volumes"
                    empty="Nothing read in the last 30 days"/>
                <MostPlayedPanel tracks={data.top_played}/>
                <ActivityPanel days={data.activity}/>
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

// in the library folders' order; checked as a set for color-blind separation and contrast on the
// dark panel, so a change here should be checked again rather than swapped for theme colors
const folderColors = { anime: "bg-[#f25912]", movie: "bg-[#8b5cf6]", music: "bg-[#4e9e54]", manga: "bg-[#4f86e8]", light_novel: "bg-[#b8860f]" };

/**
 * The disk as one bar split by what fills it: each library folder, everything else on the disk,
 * then free space. Without the disk's size the bar is the library alone
 */
function StoragePanel({ storage, history, rescanning, onRescan }) {
    const library = storage.folders.reduce((sum, f) => sum + f.bytes, 0);
    const disk = storage.disk_total ?? null;
    const used = disk != null ? disk - storage.disk_free : null;
    const whole = disk || library || 1;

    const segments = [
        ...storage.folders.map(f => ({ key: f.type, label: folderLabels[f.type] ?? f.name, bytes: f.bytes, files: f.files, color: folderColors[f.type] ?? "bg-accent" })),
        // files outside the library folders (the OS, downloads in progress, anything else on the volume)
        ...(used != null ? [{ key: "other", label: "Other files", bytes: Math.max(0, used - library), color: "bg-base-content/30" }] : []),
    ];

    return <Panel title="Storage" icon={HardDrive} meta={Format.bytes(library)}
        description={`Scanned ${Format.relativeTime(storage.scanned_at)}, kept for an hour.`}
        actions={<button className="btn btn-sm btn-ghost" onClick={onRescan} disabled={rescanning}>
            <RefreshCw size={15} className={rescanning ? "animate-spin" : ""}/> Rescan
        </button>}>
        <div className="flex justify-between gap-3 text-sm mb-2">
            <span className="text-base-content/70">{disk != null ? "Disk" : "Library"}</span>
            <span className="tabular-nums text-base-content/60 text-right">
                {disk != null
                    ? <>{Format.bytes(used)} used of {Format.bytes(disk)} · <span className="text-base-content/80">{Format.bytes(storage.disk_free)} free</span></>
                    : Format.bytes(library)}
            </span>
        </div>
        <div className="flex h-3 rounded-full bg-white/8 overflow-hidden gap-0.5" role="img"
            aria-label={segments.map(s => `${s.label} ${percent(s.bytes, whole)}`).join(", ")}>
            {segments.filter(s => s.bytes > 0).map(s => (
                // a sliver stays visible even when it rounds to nothing next to the disk
                <div key={s.key} className={`h-full ${s.color} min-w-[3px]`} style={{ width: `${(s.bytes / whole) * 100}%` }}
                    title={`${s.label}: ${Format.bytes(s.bytes)} (${percent(s.bytes, whole)})`}></div>
            ))}
        </div>

        <ul className="mt-5 grid gap-x-8 gap-y-3 sm:grid-cols-2">
            {segments.map(s => (
                <li key={s.key} className="flex items-center gap-3 text-sm min-w-0">
                    <span className={`size-2.5 shrink-0 rounded-full ${s.color}`}></span>
                    <span className="min-w-0 flex-1 truncate">
                        {s.label}
                        {s.files != null && <span className="text-base-content/40"> · {s.files.toLocaleString()} files</span>}
                    </span>
                    <span className="tabular-nums text-base-content/80 shrink-0">{Format.bytes(s.bytes)}</span>
                    <span className="tabular-nums text-base-content/45 shrink-0 w-12 text-right">{percent(s.bytes, whole)}</span>
                </li>
            ))}
        </ul>
        <p className="mt-5 text-sm text-base-content/55">
            Percentages are of the {disk != null ? "whole disk" : "library"}. Transcodes and unfinished uploads: <span className="tabular-nums text-base-content/80">{Format.bytes(storage.temp.bytes)}</span> in {storage.temp.files.toLocaleString()} files, cleared after 48 hours.
        </p>

        <div className="mt-6 pt-5 border-t border-white/5">
            {history.length >= 2
                ? <LineChart title="Library size" formatValue={Format.bytes}
                    summary={`last ${history.length} days`}
                    data={history.map(p => ({ key: p.day, label: Format.day(p.day), value: p.library_bytes }))}/>
                : <p className="text-sm text-base-content/55">The library's size is saved once a day; a chart of it shows up here from tomorrow.</p>}
        </div>
    </Panel>;
}

// part as a share of whole, "<0.1%" for something there but too small to show
function percent(part, whole) {
    if (!part) return "0%";
    const pct = (part / whole) * 100;
    if (pct < 0.1) return "<0.1%";
    return `${pct < 10 ? pct.toFixed(1) : Math.round(pct)}%`;
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

/**
 * The tracks played the most in the last 30 days
 */
function MostPlayedPanel({ tracks }) {
    return <Panel title="Most played" icon={Music} description="Last 30 days, by plays. A play counts after 30 seconds.">
        {tracks.length === 0
            ? <EmptyState compact title="Nothing played in the last 30 days"/>
            : <DataTable
                items={tracks}
                columns={[
                    { key: "title", header: "Track", primary: true, render: (val, item) => (
                        <div className="min-w-0">
                            {item.album_id
                                ? <Link to={`/albums/${item.album_id}`} className="font-medium hover:text-primary transition-colors">{val}</Link>
                                : <span className="font-medium">{val}</span>}
                            <p className="text-xs text-base-content/45 truncate">{item.artist}</p>
                        </div>
                    )},
                    { key: "users", header: "People", render: val => <span className="tabular-nums">{val}</span> },
                    { key: "plays", header: "Plays", render: val => <span className="tabular-nums">{val}</span> },
                ]}
            />}
    </Panel>;
}

/**
 * People around and tracks played per day over the last 30 days, as two charts rather than one
 * with two scales
 */
function ActivityPanel({ days }) {
    const series = key => days.map(d => ({ key: d.day, label: Format.day(d.day), value: d[key] }));
    const today = days[days.length - 1];
    const peak = days.reduce((best, d) => d.users > best.users ? d : best, days[0]);
    const plays = days.reduce((sum, d) => sum + d.plays, 0);

    return <Panel title="Activity" icon={Activity} description="Last 30 days.">
        <div className="flex flex-col gap-6">
            <ColumnChart title="People active" data={series("users")}
                summary={`today ${today.users} · peak ${peak.users} on ${Format.day(peak.day)}`}/>
            <ColumnChart title="Tracks played" data={series("plays")} color="bg-secondary"
                summary={`today ${today.plays} · ${plays.toLocaleString()} in 30 days`}/>
        </div>
    </Panel>;
}
