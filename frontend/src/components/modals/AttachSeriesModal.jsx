import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Check, Link2, Search } from "lucide-react";
import TextField from "../TextField";
import api from "../../services/api";
import { toast } from "../../stores/useToastStore";
import { errorMessage } from "../../utils/errors";
import { matchSeries, parseSeason } from "../../utils/animeMerge";

/**
 * Admin: attach an anime title the anime list leaves out of its series (Gintama's 2006 entry) to
 * that series, as one of its seasons. The title is merged into the series' one, carrying
 * everyone's progress, status, ratings and wishlist over, and later saves of it land there too
 * @param {boolean} open whether it shows; the series are only loaded while it does
 * @param {function} onClose called when it closes
 * @param {object} item the title: id, title
 * @param {function} onAttached called with the id of the series' title once merged
 */
export default function AttachSeriesModal({ open, onClose, item, onAttached }) {
    const dialogRef = useRef(null);

    useEffect(() => {
        const dialog = dialogRef.current;
        if (open && dialog && !dialog.open) dialog.showModal();
    }, [open]);

    return createPortal(
        <dialog className="modal modal-bottom sm:modal-middle" ref={dialogRef} onClose={onClose}>
            {open && <AttachForm dialogRef={dialogRef} item={item} onAttached={onAttached}/>}
            <form method="dialog" className="modal-backdrop"><button aria-label="Close">close</button></form>
        </dialog>,
        document.body
    );
}

function AttachForm({ dialogRef, item, onAttached }) {
    const [series, setSeries] = useState(null);
    const [loadError, setLoadError] = useState(null);
    const [query, setQuery] = useState("");
    const [picked, setPicked] = useState(null);
    const [seasonText, setSeasonText] = useState("1");
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState("");

    useEffect(() => {
        api.get("/admin/anime-series")
            .then(res => setSeries(res.data ?? []))
            .catch(err => setLoadError(err));
    }, []);

    const season = parseSeason(seasonText);
    const shown = matchSeries(series, query, item.id).slice(0, 50);

    function attach(e) {
        e.preventDefault();
        if (!picked || season === null) return;
        setBusy(true);
        setError("");
        api.post(`/admin/anime/${item.id}/series`, { tvdb_id: picked.tvdb_id, season })
            .then(res => {
                toast.success(`Attached to ${picked.title}`);
                dialogRef.current?.close();
                onAttached?.(res.data?.keep_id ?? picked.id);
            })
            .catch(err => setError(errorMessage(err, "Could not attach it")))
            .finally(() => setBusy(false));
    }

    return <form onSubmit={attach} className="modal-box flex flex-col gap-4 max-h-[85dvh]">
        <h2 className="text-xl font-semibold min-w-0 break-words">Attach “{item.title}” to a series</h2>
        <p className="text-sm text-base-content/60">
            For an entry the anime list doesn't place in its series. It's merged into the series' title as
            the season you pick, with everyone's progress, ratings and wishlist, and saving it again later lands there too.
        </p>

        <TextField label="Series" icon={Search} value={query} onChange={e => setQuery(e.target.value)}
            placeholder="Search the library's series" autoComplete="off"/>

        {loadError ? (
            <p className="text-sm text-error" role="alert">{errorMessage(loadError, "Could not load the series")}</p>
        ) : series === null ? (
            <div className="flex flex-col gap-2" aria-busy="true">
                {[0, 1, 2].map(i => <div key={i} className="skeleton h-10 w-full"/>)}
            </div>
        ) : shown.length === 0 ? (
            <p className="text-sm text-base-content/60">No series match. Only titles that stand for a whole series (from Sonarr) are listed.</p>
        ) : (
            <ul className="flex flex-col gap-1 overflow-y-auto -mx-2 px-2 min-h-0" role="radiogroup" aria-label="Series">
                {shown.map(s => {
                    const on = picked?.id === s.id;
                    return <li key={s.id}>
                        <button type="button" role="radio" aria-checked={on} onClick={() => setPicked(s)}
                            className={`w-full flex items-center gap-3 rounded-field px-3 py-2 text-left text-sm transition-colors ${on ? "bg-primary/15" : "hover:bg-white/5"}`}>
                            <span className="flex-1 min-w-0 truncate">{s.title}</span>
                            {on && <Check size={16} className="text-primary shrink-0"/>}
                        </button>
                    </li>
                })}
            </ul>
        )}

        <TextField label="As season" inputMode="numeric" value={seasonText} onChange={e => setSeasonText(e.target.value)}
            hint="0 for specials" aria-invalid={season === null}/>

        {error && <p role="alert" className="text-sm text-error break-words">{error}</p>}

        <div className="flex justify-end gap-2">
            <button type="button" className="btn btn-ghost" onClick={() => dialogRef.current?.close()}>Cancel</button>
            <button type="submit" className="btn btn-primary" disabled={busy || !picked || season === null}>
                {busy ? <span className="loading loading-spinner loading-xs"/> : <Link2 size={16}/>}
                Attach and merge
            </button>
        </div>
    </form>;
}
