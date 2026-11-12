import { useState } from "react";
import { createPortal } from "react-dom";
import { PartyPopper, Star } from "lucide-react";
import WheelPicker from "../WheelPicker";
import { ratingWord } from "../../utils/ratingPrompt";

/**
 * Rate a title from 1 to 10, or clear its rating
 * @param {object} dialogRef ref for the <dialog>
 * @param {string} title the title being rated
 * @param {number} rating its rating now, null for none
 * @param {boolean} prompt asked because it was just completed, not opened by the user
 * @param {function} onSave called with the rating (null clears it); returns a promise, the dialog
 *  closes when it resolves and stays open (for a retry) when it rejects
 * @param {function} onDismiss called when it's closed without saving
 */
export default function RatingModal({ dialogRef, title, rating, prompt = false, onSave, onDismiss }) {
    const [score, setScore] = useState(rating ?? 0);
    const [busy, setBusy] = useState(false);
    // saving closes the dialog too, which mustn't count as dismissing it
    const [saved, setSaved] = useState(false);

    // start over from the saved rating whenever it changes
    const [shownRating, setShownRating] = useState(rating);
    if (rating !== shownRating) {
        setShownRating(rating);
        setScore(rating ?? 0);
    }

    function save() {
        setBusy(true);
        setSaved(true);
        Promise.resolve(onSave?.(score === 0 ? null : score))
            .then(() => dialogRef.current?.close())
            .catch(() => setSaved(false))
            .finally(() => setBusy(false));
    }

    function handleClose() {
        if (!saved) onDismiss?.();
        setSaved(false);
        setScore(rating ?? 0);
    }

    const word = ratingWord(score);

    return createPortal(
        // React passes events up through the portal, and a card's dialog sits inside its link
        <dialog className="modal modal-bottom sm:modal-middle" ref={dialogRef} onClose={handleClose} onClick={e => e.stopPropagation()}>
            <div className="modal-box flex flex-col items-center gap-4 text-center">
                <div>
                    {prompt
                        ? <p className="flex items-center justify-center gap-1.5 text-xs font-medium uppercase tracking-wider text-primary mb-1">
                            <PartyPopper size={14}/> Finished
                        </p>
                        : <p className="flex items-center justify-center gap-1.5 text-xs font-medium uppercase tracking-wider text-primary mb-1">
                            <Star size={14}/> Your rating
                        </p>}
                    <h2 className="text-xl font-semibold text-balance break-words">
                        {title ? (prompt ? `How was ${title}?` : title) : (prompt ? "How was it?" : "Rate it")}
                    </h2>
                </div>
                <WheelPicker value={score} onChange={setScore} min={0} max={10} unsetValue={0} label="Rating out of 10"/>
                <p className="h-5 text-sm text-base-content/70" aria-live="polite">
                    {word ? <><span className="font-semibold text-base-content">{score}</span>/10 · {word}</> : "No rating"}
                </p>
                <div className="flex w-full gap-2 sm:justify-end">
                    <button type="button" className="btn btn-ghost flex-1 sm:flex-none" onClick={() => dialogRef.current?.close()} disabled={busy}>
                        {prompt ? "Not now" : "Cancel"}
                    </button>
                    <button type="button" className="btn btn-primary flex-1 sm:flex-none" onClick={save} disabled={busy || (score === 0 && rating == null)}>
                        {busy && <span className="loading loading-spinner loading-xs"/>}
                        {score === 0 && rating != null ? "Clear rating" : "Save"}
                    </button>
                </div>
            </div>
            <form method="dialog" className="modal-backdrop"><button disabled={busy}>close</button></form>
        </dialog>,
        document.body
    );
}
