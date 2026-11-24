import { useEffect, useState } from "react";
import { Link2, Trash2 } from "lucide-react";
import api from "../../services/api";
import { Panel } from "../../components/DataTable";
import { toast } from "../../stores/useToastStore";
import { errorMessage } from "../../utils/errors";
import { seasonLabel } from "../../utils/animeMerge";

/**
 * Admin section listing the anime entries attached to a series by hand (from a title's "Attach
 * to a series…"), with a way to forget one. Forgetting only changes later saves and syncs: a
 * title already merged into the series stays merged
 */
export default function SeriesLinksSection() {
    const [links, setLinks] = useState(null);
    const [error, setError] = useState("");
    const [busy, setBusy] = useState(null);

    useEffect(() => {
        api.get("/admin/anime-series-links")
            .then(res => setLinks(res.data ?? []))
            .catch(err => setError(errorMessage(err, "Could not load the series links")));
    }, []);

    function remove(link) {
        setBusy(link.anilist_id);
        api.delete(`/admin/anime-series-links/${link.anilist_id}`)
            .then(() => {
                setLinks(prev => prev.filter(l => l.anilist_id !== link.anilist_id));
                toast.success("Link removed");
            })
            .catch(err => toast.error(err, "Could not remove the link"))
            .finally(() => setBusy(null));
    }

    return (
        <Panel
            title="Anime attached to a series"
            icon={Link2}
            meta={links ? links.length : null}
            description="Entries the anime list leaves out of their series, attached by hand from a title's menu. Removing a link only changes later saves and syncs; titles already merged stay merged."
        >
            {error && <p role="alert" className="text-error text-sm">{error}</p>}
            {!error && links === null && <div className="skeleton h-10 w-full" aria-busy="true"/>}
            {links?.length === 0 && <p className="text-sm text-base-content/55">Nothing attached by hand.</p>}
            {links?.length > 0 && <ul className="flex flex-col gap-1">
                {links.map(l => <li key={l.anilist_id} className="flex items-center gap-3 rounded-field px-3 py-2.5 bg-white/[0.03] ring-1 ring-white/5">
                    <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium break-words">{l.title || `AniList ${l.anilist_id}`}</p>
                        <p className="text-xs text-base-content/55 break-words">
                            {seasonLabel({ season: l.season })} of <span className="text-base-content/80">{l.series || `TVDB ${l.tvdb_id}`}</span>
                        </p>
                    </div>
                    <button className="btn btn-sm btn-ghost btn-square" aria-label={`Remove the link for ${l.title || l.anilist_id}`}
                        disabled={busy !== null} onClick={() => remove(l)}>
                        {busy === l.anilist_id ? <span className="loading loading-spinner loading-xs"/> : <Trash2 size={15}/>}
                    </button>
                </li>)}
            </ul>}
        </Panel>
    );
}
