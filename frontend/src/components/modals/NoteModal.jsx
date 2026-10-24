import { useState } from "react";
import { createPortal } from "react-dom";
import TextField from "../TextField";
import { errorMessage } from "../../utils/errors";

/**
 * Do something with an optional note attached, like submitting a playlist with a message for the
 * admins or turning one down with a reason
 * @param {object} dialogRef ref for the <dialog>
 * @param {string} title the heading
 * @param {any} intro a line under the heading saying what happens
 * @param {string} label the note field's label
 * @param {string} confirmLabel the button that goes ahead
 * @param {string} tone "error" for a button that turns something down
 * @param {function} onSave called with the note; returns a promise, the dialog closes when it
 *  resolves and shows the error when it rejects
 */
export default function NoteModal({ dialogRef, title, intro, label = "Note (optional)", confirmLabel, tone = "primary", onSave }) {
    const [note, setNote] = useState("");
    const [error, setError] = useState("");
    const [busy, setBusy] = useState(false);

    function save(e) {
        e.preventDefault();
        setBusy(true);
        setError("");
        Promise.resolve(onSave(note.trim()))
            .then(() => { dialogRef.current?.close(); setNote(""); })
            .catch(err => setError(errorMessage(err, "Something went wrong")))
            .finally(() => setBusy(false));
    }

    return createPortal(
        <dialog className="modal modal-bottom sm:modal-middle" ref={dialogRef}>
            <form className="modal-box flex flex-col gap-4" onSubmit={save}>
                <h2 className="text-xl font-semibold break-words">{title}</h2>
                {intro && <p className="text-sm text-base-content/70">{intro}</p>}
                <TextField label={label} value={note} onChange={e => setNote(e.target.value)} maxLength={500}/>
                {error && <p className="text-sm text-error" role="alert">{error}</p>}
                <div className="flex gap-2 justify-end">
                    <button type="button" className="btn btn-ghost" onClick={() => dialogRef.current?.close()} disabled={busy}>Cancel</button>
                    <button type="submit" className={`btn ${tone === "error" ? "btn-error" : "btn-primary"}`} disabled={busy}>
                        {busy && <span className="loading loading-spinner loading-xs"/>}
                        {confirmLabel}
                    </button>
                </div>
            </form>
            <form method="dialog" className="modal-backdrop"><button aria-label="Close" disabled={busy}>close</button></form>
        </dialog>,
        document.body
    );
}
