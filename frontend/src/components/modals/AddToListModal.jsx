import { useEffect, useState } from "react";
import api from "../../services/api";
import WheelPicker from "../WheelPicker";
import Format from "../../utils/format";
import { hasParts, partsLabel, progressRange } from "../../utils/progress";

/**
 * Add a title to the user's list or edit its entry
 * @param {function} onConfirm called with { status, score, progress }; progress is undefined
 *  unless it was changed, so chapters read out of order aren't redone
 * @param {object} initialValues the user's list entry, when the title is already on their list
 * @param {boolean} isLibraryItem the title is in the library (no wishlist option then)
 * @param {function} onRemove takes the title off the user's list, shown for titles on it
 * @param {function} onDelete admins: opens the permanent delete confirmation
 * @param {function} onAddToCollection opens the playlist / collection picker, for titles in the library
 * @param {string} mediaId the title's id in the library, to load the user's progress through it
 * @param {number} openCount goes up each time the dialog opens, which reloads that progress
 */
export default function AddToListModal({item, onConfirm, dialogRef, initialValues, isLibraryItem, onRemove, onDelete, onAddToCollection, mediaId, openCount = 0}) {
    const type = item.type || item.media_type;
    const [status, setStatus] = useState(initialValues?.status ?? 'planned');
    const [score, setScore] = useState(initialValues?.rating ?? 0);
    // null until the user moves the wheel, so it follows the numbers as they load
    const [progress, setProgress] = useState(null);
    // how far through the title the server says the user is, fetched each time the dialog opens
    const [fetched, setFetched] = useState(null);

    // start over from the entry whenever a different one comes in
    const [shownValues, setShownValues] = useState(initialValues);
    if (initialValues !== shownValues) {
        setShownValues(initialValues);
        setStatus(initialValues?.status ?? 'planned');
        setScore(initialValues?.rating ?? 0);
        setProgress(null);
    }

    // titles in the library, on the user's list or not (Continue and library rows don't carry
    // the numbers), get their progress and length from the server
    const loadsProgress = !!mediaId && hasParts(type);
    useEffect(() => {
        if (!loadsProgress || openCount === 0) return;
        let current = true;
        api.get(`/me/progress/${mediaId}`)
            .then(res => { if (current) { setFetched(res.data); setProgress(null); } })
            .catch(() => {});
        return () => { current = false; };
    }, [loadsProgress, mediaId, openCount]);

    // episodes, chapters or volumes, up to as many as the title's source says it has, or the
    // files when nobody knows
    const info = fetched ?? initialValues;
    const range = progressRange(info, item);
    const showProgress = hasParts(type) && range.max > 0;
    const startProgress = info?.progress_count ?? 0;
    const shownProgress = progress ?? startProgress;
    const progressChanged = progress !== null && progress !== startProgress;

    function statusOptions(type) {
        const withWishlist = !isLibraryItem;
        switch(type) {
            case "music_album": return ["completed", "planned"]
            case "movie": return ["completed", "current", ...(withWishlist ? ["wishlist"] : [])]
            default: return ["completed", "current", "dropped", "planned", ...(withWishlist ? ["wishlist"] : [])]
        }
    }

    return <dialog className="modal modal-bottom sm:modal-middle" ref={dialogRef} onClick={e => { e.stopPropagation(); e.preventDefault(); }}>
        <div className="modal-box flex flex-col gap-4">
            <h2 className="text-xl font-semibold">{item.title || item.media_title}</h2>
            <section>
                <h4 className="text-xs font-medium text-base-content/80 uppercase tracking-wide mb-1">Status</h4>
                <div className="flex flex-wrap gap-2">
                    {statusOptions(type).map(option => (
                        <button
                            key={option}
                            type="button"
                            className={`btn btn-sm ${option === status ? "btn-primary" : "btn-ghost bg-white/5 hover:bg-white/10"}`}
                            onClick={() => setStatus(option)}
                        >
                            {Format.statusLabel(option, type)}
                        </button>
                    ))}
                </div>
            </section>
            {showProgress && status !== "completed" && (
                <section>
                    <h4 className="text-xs font-medium text-base-content/80 uppercase tracking-wide mb-1">
                        {partsLabel(type)}
                        {range.known && <span className="normal-case tracking-normal text-base-content/50"> · of {range.max}</span>}
                    </h4>
                    <WheelPicker value={shownProgress} onChange={setProgress} max={range.max} label={partsLabel(type)}/>
                </section>
            )}
            <section>
                <h4 className="text-xs font-medium text-base-content/80 uppercase tracking-wide mb-1">Score</h4>
                <WheelPicker value={score} onChange={setScore} max={10} unsetValue={0} label="Score out of 10"/>
            </section>
            <section className="flex flex-wrap gap-2 pt-2">
                <button className="btn btn-primary" onClick={() => {
                    onConfirm({ status, score, progress: showProgress && progressChanged ? progress : undefined });
                    dialogRef.current.close();
                }}>Confirm</button>
                <button className="btn btn-ghost" onClick={() => dialogRef.current.close()}>Cancel</button>
                {initialValues && onRemove && (
                    <button className="btn btn-ghost text-error ml-auto" onClick={() => { dialogRef.current.close(); onRemove(); }}>
                        Remove from list
                    </button>
                )}
            </section>
            {(onAddToCollection || onDelete) && (
                <section className="flex flex-wrap gap-x-6 gap-y-2 border-t border-white/5 pt-3">
                    {onAddToCollection && <button className="link link-hover text-sm text-primary" onClick={() => { dialogRef.current.close(); onAddToCollection(); }}>
                        Add to a playlist or collection…
                    </button>}
                    {onDelete && <button className="link link-error text-sm" onClick={() => { dialogRef.current.close(); onDelete(); }}>
                        Delete from server…
                    </button>}
                </section>
            )}
        </div>
    </dialog>
}
