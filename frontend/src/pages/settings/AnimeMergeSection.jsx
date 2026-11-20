import { useState } from "react";
import api from "../../services/api";
import { errorMessage } from "../../utils/errors";
import { Combine, Search } from "lucide-react";
import { Panel } from "../../components/DataTable";
import StatusPill from "../../components/StatusPill";
import { toast } from "../../stores/useToastStore";
import { seasonLabel } from "../../utils/animeMerge";

/**
 * Admin section that finds anime seasons saved as titles of their own (before titles followed
 * Sonarr) and folds them into their series. Finding is a dry run; only the series ticked merge
 */
export default function AnimeMergeSection() {
    const [groups, setGroups] = useState(null);
    const [picked, setPicked] = useState(new Set());
    const [results, setResults] = useState({});
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState("");

    function find() {
        setBusy(true);
        setError("");
        setResults({});
        api.get("/admin/anime-merges")
            .then(res => {
                const found = res.data ?? [];
                setGroups(found);
                setPicked(new Set(found.map(g => g.tvdb_id)));
            })
            .catch(err => setError(errorMessage(err, "Could not look for duplicate titles")))
            .finally(() => setBusy(false));
    }

    function merge() {
        setBusy(true);
        setError("");
        api.post("/admin/anime-merges", { tvdb_ids: [...picked] })
            .then(res => {
                const byId = Object.fromEntries((res.data ?? []).map(r => [r.tvdb_id, r]));
                setResults(byId);
                const failed = (res.data ?? []).filter(r => r.error).length;
                if (failed) toast.error(null, `${failed} couldn't be merged`);
                else toast.success("Merged");
                setPicked(new Set());
            })
            .catch(err => setError(errorMessage(err, "Could not merge")))
            .finally(() => setBusy(false));
    }

    function toggle(id) {
        const next = new Set(picked);
        next.has(id) ? next.delete(id) : next.add(id);
        setPicked(next);
    }

    const pending = (groups ?? []).filter(g => !results[g.tvdb_id] || results[g.tvdb_id].error);

    return (
        <Panel
            title="Merge anime seasons"
            icon={Combine}
            meta={groups ? groups.length : null}
            description="Seasons saved as titles of their own, from AniList or a MyAnimeList import, folded into their series the way Sonarr keeps it. Everyone's progress, status and ratings carry over."
            actions={<>
                <button className="btn btn-sm" onClick={find} disabled={busy}>
                    {busy && !groups ? <span className="loading loading-spinner loading-xs"/> : <Search size={15}/>}
                    {groups ? "Look again" : "Find"}
                </button>
                {groups?.length > 0 && <button className="btn btn-sm btn-primary" onClick={merge} disabled={busy || picked.size === 0}>
                    {busy && groups ? <span className="loading loading-spinner loading-xs"/> : <Combine size={15}/>}
                    Merge {picked.size}
                </button>}
            </>}
        >
            {error && <p role="alert" className="text-error text-sm mb-3">{error}</p>}
            {groups && groups.length === 0 && <p className="text-sm text-base-content/55">Every series has one title.</p>}
            {groups?.length > 0 && <ul className="flex flex-col gap-1">
                {groups.map(g => {
                    const result = results[g.tvdb_id];
                    const done = result && !result.error;
                    return <li key={g.tvdb_id} className="flex items-start gap-3 rounded-field px-3 py-2.5 bg-white/[0.03] ring-1 ring-white/5">
                        <input type="checkbox" className="checkbox checkbox-sm mt-0.5" aria-label={`Merge ${g.series}`}
                            checked={picked.has(g.tvdb_id)} disabled={done || busy} onChange={() => toggle(g.tvdb_id)}/>
                        <div className="flex-1 min-w-0">
                            <p className="text-sm font-medium">{g.series}</p>
                            <p className="text-xs text-base-content/55 break-words">
                                Keeps <span className="text-base-content/80">{g.keep.title}</span> ({seasonLabel(g.keep)}),
                                folds in {g.drop.map((d, i) => <span key={d.id}>
                                    {i > 0 && ", "}<span className="text-base-content/80">{d.title}</span> ({seasonLabel(d)})
                                </span>)}
                            </p>
                            {result?.error && <p className="text-xs text-error break-words">{result.error}</p>}
                        </div>
                        {done && <StatusPill label={`${result.merged} merged`} tone="success"/>}
                    </li>
                })}
            </ul>}
            {groups?.length > 0 && pending.length === 0 && <p className="text-xs text-base-content/45 mt-3">All merged.</p>}
        </Panel>
    );
}
