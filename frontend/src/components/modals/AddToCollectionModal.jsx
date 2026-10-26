import { useState } from "react";
import { createPortal } from "react-dom";
import { Check, Plus } from "lucide-react";
import { CollectionCover } from "../cards";
import TextField from "../TextField";
import { addToCollection, createCollection } from "../../services/collectionsService";
import { addBody, canHold, itemKind, itemTitle } from "../../utils/collections";
import { toast } from "../../stores/useToastStore";
import { errorMessage } from "../../utils/errors";

/**
 * Pick the playlists or collections to add something to, or make a new one for it. Only the
 * ones that can take it are offered, and the ones that already have it are ticked
 * @param {object} dialogRef ref for the <dialog>
 * @param {object} item what's being added, as for utils/collections itemKind, addBody and
 *  itemTitle
 * @param {object[]|null} options the user's playlists and collections, null while they load
 * @param {any} loadError why they couldn't be loaded
 * @param {function} onAdded called after something was added, to reload what shows it. Pages
 *  load again on their own when opened, so only what's on screen needs it
 */
export default function AddToCollectionModal({ dialogRef, item, options, loadError, onAdded }) {
    const [added, setAdded] = useState({});
    const [busy, setBusy] = useState(null);
    const [newName, setNewName] = useState("");
    const [newKind, setNewKind] = useState(null);

    if (!item) return null;
    const kind = itemKind(item);
    const title = itemTitle(item);
    const kinds = ["playlist", "collection"].filter(k => canHold(k, kind));
    const shown = (options ?? []).filter(c => !c.smart && kinds.includes(c.kind) && c.id !== item.id);

    function report(collection, res) {
        const { added: n = 0, already = 0 } = res.data ?? {};
        if (n === 0) toast.success(`Already in ${collection.name}`);
        else if (kind === "album" && collection.kind === "playlist") {
            toast.success(`Added ${n} ${n === 1 ? "track" : "tracks"} to ${collection.name}${already ? ` (${already} already there)` : ""}`);
        } else toast.success(`Added to ${collection.name}`);
        setAdded(prev => ({ ...prev, [collection.id]: true }));
        onAdded?.();
    }

    function add(collection) {
        setBusy(collection.id);
        addToCollection(collection.id, addBody(item))
            .then(res => report(collection, res))
            .catch(err => toast.error(err, `Could not add it to ${collection.name}`))
            .finally(() => setBusy(null));
    }

    function create(e) {
        e.preventDefault();
        const k = newKind ?? kinds[0];
        if (!newName.trim()) return;
        setBusy("new");
        createCollection(k, newName.trim())
            .then(res => {
                const collection = { id: res.data.id, kind: k, name: newName.trim() };
                setNewName("");
                return addToCollection(collection.id, addBody(item)).then(r => report(collection, r));
            })
            .catch(err => toast.error(err, `Could not make the ${k}`))
            .finally(() => setBusy(null));
    }

    const what = kinds.length === 2 ? "playlist or collection" : kinds[0] ?? "collection";

    // React passes events up through a portal to whatever renders it, which is often a card's
    // link or play button: clicks stop here so picking a playlist doesn't also open the title.
    // Not cancelled, or the Create button couldn't submit its form
    return createPortal(
        <dialog className="modal modal-bottom sm:modal-middle" ref={dialogRef} onClose={() => setAdded({})}
            onClick={e => e.stopPropagation()}>
            <div className="modal-box flex flex-col gap-4 max-h-[85dvh]">
                <h2 className="text-xl font-semibold min-w-0 break-words">Add “{title}” to…</h2>

                {loadError ? (
                    <p className="text-sm text-error" role="alert">{errorMessage(loadError, "Could not load your playlists and collections")}</p>
                ) : options === null ? (
                    <div className="flex flex-col gap-2" aria-busy="true">
                        {[0, 1, 2].map(i => <div key={i} className="skeleton h-12 w-full"/>)}
                    </div>
                ) : shown.length === 0 ? (
                    <p className="text-sm text-base-content/60">You don't have a {what} for this yet. Make one below.</p>
                ) : (
                    <ul className="flex flex-col gap-1 overflow-y-auto -mx-2 px-2">
                        {shown.map(c => {
                            const has = c.contains || added[c.id];
                            return <li key={c.id}>
                                <button type="button" disabled={busy !== null || has} onClick={() => add(c)}
                                    className="w-full flex items-center gap-3 rounded-field p-2 text-left transition-colors hover:bg-white/5 disabled:hover:bg-transparent">
                                    <CollectionCover covers={c.covers} kind={c.kind} className="size-11 shrink-0 rounded-md overflow-hidden"/>
                                    <span className="flex-1 min-w-0">
                                        <span className="block truncate text-sm font-medium">{c.name}</span>
                                        <span className="block text-xs text-base-content/50">
                                            {c.kind === "collection" ? "Collection" : "Playlist"} · {c.item_count} {c.kind === "collection" ? "items" : "tracks"}
                                        </span>
                                    </span>
                                    {busy === c.id
                                        ? <span className="loading loading-spinner loading-xs"/>
                                        : has ? <Check size={18} className="text-primary" aria-label="Already in it"/> : <Plus size={18} className="text-base-content/50"/>}
                                </button>
                            </li>
                        })}
                    </ul>
                )}

                {kinds.length > 0 && <form onSubmit={create} className="flex flex-col gap-2 border-t border-white/5 pt-3">
                    <div className="flex items-end gap-2">
                        <div className="flex-1 min-w-0">
                            <TextField label={`New ${kinds.length === 2 ? (newKind ?? kinds[0]) : kinds[0]}`} value={newName}
                                onChange={e => setNewName(e.target.value)} maxLength={100} placeholder="Name"/>
                        </div>
                        <button type="submit" className="btn btn-primary h-11" disabled={busy !== null || !newName.trim()}>
                            {busy === "new" && <span className="loading loading-spinner loading-xs"/>} Create
                        </button>
                    </div>
                    {kinds.length === 2 && <div className="flex gap-2 text-sm" role="radiogroup" aria-label="Make a">
                        {kinds.map(k => <button key={k} type="button" role="radio" aria-checked={(newKind ?? kinds[0]) === k}
                            className={`btn btn-xs ${(newKind ?? kinds[0]) === k ? "btn-primary" : "btn-ghost bg-white/5"}`}
                            onClick={() => setNewKind(k)}>{k === "playlist" ? "Playlist" : "Collection"}</button>)}
                    </div>}
                </form>}

                <div className="flex justify-end">
                    <button type="button" className="btn btn-ghost" onClick={() => dialogRef.current?.close()}>Done</button>
                </div>
            </div>
            <form method="dialog" className="modal-backdrop"><button aria-label="Close">close</button></form>
        </dialog>,
        document.body
    );
}
