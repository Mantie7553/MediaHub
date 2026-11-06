import { useState } from "react";
import { createPortal } from "react-dom";
import { Check, Link2, ListOrdered, Unlink } from "lucide-react";
import SearchField from "../SearchField";
import { linkSuwayomi, unlinkSuwayomi } from "../../services/mediaService";
import { toast } from "../../stores/useToastStore";
import { errorMessage } from "../../utils/errors";

/**
 * Admin: pick the Suwayomi source title a manga's missing chapters are downloaded from. Exact
 * title matches are listed first. Opened, and its search run, by hooks/useLinkSource
 * @param {object} dialogRef ref for the <dialog>
 * @param {object} manga { id, title }
 * @param {object|null} link the current link, { source_name, manga_id, title }
 * @param {string} query the search text
 * @param {function} onQueryChange called with new search text
 * @param {function} onSearch runs the search
 * @param {object|null} search the last search's answer
 * @param {boolean} searching a search is running
 * @param {any} error why the search failed
 * @param {object} looked chapters looked up after the search, by Suwayomi title id:
 *  { loading } | { chapters } | { error }
 * @param {function} onLookUp looks up a title's chapters, by its Suwayomi id
 * @param {function} onLinked called with the new link
 * @param {function} onUnlinked called after the link is removed
 */
export default function LinkSourceModal({ dialogRef, manga, link, query, onQueryChange, onSearch, search, searching, error, looked, onLookUp, onLinked, onUnlinked }) {
    const [busy, setBusy] = useState(null);

    function pick(result) {
        setBusy(result.id);
        linkSuwayomi(manga.id, result.id)
            .then(res => {
                onLinked(res.data);
                dialogRef.current?.close();
            })
            .catch(err => toast.error(err, "Could not link that title"))
            .finally(() => setBusy(null));
    }

    function unlink() {
        setBusy("unlink");
        unlinkSuwayomi(manga.id)
            .then(() => { onUnlinked(); toast.success("Unlinked. Chapters already downloaded stay"); })
            .catch(err => toast.error(err, "Could not unlink it"))
            .finally(() => setBusy(null));
    }

    const results = search?.results ?? [];
    const isLinked = r => link && link.manga_id === r.id;
    // a result's chapters: checked by the search, or looked up since. undefined when neither
    const chaptersOf = r => r.checked ? { chapters: r.chapters } : looked[r.id];
    const linkInfo = link ? looked[link.manga_id] : undefined;

    return createPortal(
        <dialog className="modal modal-bottom sm:modal-middle" ref={dialogRef} onClick={e => e.stopPropagation()}>
            <div className="modal-box flex flex-col gap-4 max-h-[85dvh]">
                <div>
                    <h2 className="text-xl font-semibold min-w-0 break-words">Chapter source for “{manga.title}”</h2>
                    <p className="text-sm text-base-content/60 mt-1">
                        Chapters MangaDex doesn't host are downloaded from the title you pick here.
                    </p>
                </div>

                {link && <div className="flex items-center gap-3 rounded-field surface p-3">
                    <Link2 size={18} className="text-primary shrink-0"/>
                    <span className="flex-1 min-w-0">
                        <span className="block truncate text-sm font-medium">{link.title}</span>
                        <span className="block text-xs text-base-content/50">
                            Linked · {link.source_name}
                            {linkInfo && <> · <ChapterInfo info={linkInfo}/></>}
                        </span>
                    </span>
                    <button type="button" className="btn btn-ghost btn-sm" disabled={busy !== null} onClick={unlink}>
                        {busy === "unlink" ? <span className="loading loading-spinner loading-xs"/> : <Unlink size={16}/>} Unlink
                    </button>
                </div>}

                <SearchField value={query} onChange={onQueryChange} onSubmit={onSearch} searching={searching}
                    placeholder="Search the installed sources"/>

                {error ? (
                    <p className="text-sm text-error" role="alert">{errorMessage(error, "Could not search Suwayomi")}</p>
                ) : searching ? (
                    <div className="flex flex-col gap-2" aria-busy="true">
                        <p className="text-sm text-base-content/60">Searching the sources, this can take a little while…</p>
                        {[0, 1, 2].map(i => <div key={i} className="skeleton h-12 w-full"/>)}
                    </div>
                ) : search && (
                    <>
                        {search.sources === 0 && <p className="text-sm text-base-content/60">
                            No English sources are installed in Suwayomi yet. Install one from its web UI first.
                        </p>}
                        {search.failed.length > 0 && <p className="text-xs text-warning">
                            {search.failed.map(f => f.source).join(", ")} didn't answer, so {search.failed.length === 1 ? "its" : "their"} results are missing.
                        </p>}
                        {search.sources > 0 && results.length === 0 && <p className="text-sm text-base-content/60">
                            Nothing found for “{search.query}”. Try a shorter or alternative title.
                        </p>}
                        {results.length > 0 && <ul className="flex flex-col gap-1 overflow-y-auto -mx-2 px-2">
                            {results.map(r => {
                                const info = chaptersOf(r);
                                return <li key={`${r.source_id}-${r.id}`} className="flex items-center gap-1">
                                    <button type="button" disabled={busy !== null || isLinked(r)} onClick={() => pick(r)}
                                        className="flex-1 min-w-0 flex items-center gap-3 rounded-field p-2 text-left transition-colors hover:bg-white/5 disabled:hover:bg-transparent">
                                        <span className="flex-1 min-w-0">
                                            <span className="flex items-center gap-2 min-w-0">
                                                <span className="truncate text-sm font-medium">{r.title}</span>
                                                {r.most_recent && <span className="badge badge-primary badge-sm shrink-0">Most recent</span>}
                                            </span>
                                            <span className="block text-xs text-base-content/50">
                                                {r.source_name}{r.exact && <span className="text-primary"> · exact match</span>}
                                                {info && <> · <ChapterInfo info={info}/></>}
                                            </span>
                                        </span>
                                        {busy === r.id
                                            ? <span className="loading loading-spinner loading-xs"/>
                                            : isLinked(r) ? <Check size={18} className="text-primary" aria-label="Linked"/> : <Link2 size={18} className="text-base-content/50"/>}
                                    </button>
                                    {!info && <button type="button" className="btn btn-ghost btn-sm btn-square shrink-0" onClick={() => onLookUp(r.id)}
                                        aria-label={`Show the latest chapter of ${r.title} on ${r.source_name}`} title="Show latest chapter">
                                        <ListOrdered size={16}/>
                                    </button>}
                                </li>
                            })}
                        </ul>}
                    </>
                )}

                <div className="flex justify-end">
                    <button type="button" className="btn btn-ghost" onClick={() => dialogRef.current?.close()}>Done</button>
                </div>
            </div>
            <form method="dialog" className="modal-backdrop"><button aria-label="Close">close</button></form>
        </dialog>,
        document.body
    );
}

/**
 * How far a source's copy of a title goes, as "Latest ch. 213 · 213 chapters"
 * @param {object} info { loading } | { chapters: { latest, count } | null } | { error }
 */
function ChapterInfo({ info }) {
    if (info.loading) return <span className="loading loading-dots loading-xs align-middle" aria-label="Looking up chapters"/>;
    if (info.error) return <span className="text-warning">chapters didn't load</span>;
    if (!info.chapters) return <span className="text-warning">no numbered chapters</span>;
    const { latest, count } = info.chapters;
    return <span>Latest ch. {latest} · {count} {count === 1 ? "chapter" : "chapters"}</span>;
}
