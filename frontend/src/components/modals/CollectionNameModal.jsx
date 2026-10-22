import { useState } from "react";
import { createPortal } from "react-dom";
import TextField from "../TextField";
import { errorMessage } from "../../utils/errors";

/**
 * Name a new playlist or collection, or rename one
 * @param {object} dialogRef ref for the <dialog>
 * @param {string} kind "playlist" or "collection"
 * @param {object} initial { name, description } when renaming
 * @param {function} onSave called with { name, description }; returns a promise, the dialog
 *  closes when it resolves and shows the error when it rejects
 */
export default function CollectionNameModal({ dialogRef, kind, initial, onSave }) {
    const [name, setName] = useState(initial?.name ?? "");
    const [description, setDescription] = useState(initial?.description ?? "");
    const [error, setError] = useState("");
    const [busy, setBusy] = useState(false);
    // start over from what's saved each time a different one comes in
    const [shown, setShown] = useState(initial);
    if (initial !== shown) {
        setShown(initial);
        setName(initial?.name ?? "");
        setDescription(initial?.description ?? "");
        setError("");
    }

    const noun = kind === "collection" ? "collection" : "playlist";

    function save(e) {
        e.preventDefault();
        if (!name.trim()) { setError("Give it a name"); return; }
        setBusy(true);
        setError("");
        Promise.resolve(onSave({ name: name.trim(), description: description.trim() }))
            .then(() => {
                dialogRef.current?.close();
                if (!initial) { setName(""); setDescription(""); }
            })
            .catch(err => setError(errorMessage(err, `Could not save the ${noun}`)))
            .finally(() => setBusy(false));
    }

    return createPortal(
        <dialog className="modal modal-bottom sm:modal-middle" ref={dialogRef}>
            <form className="modal-box flex flex-col gap-4" onSubmit={save}>
                <h2 className="text-xl font-semibold">{initial ? `Rename ${noun}` : `New ${noun}`}</h2>
                <TextField label="Name" value={name} onChange={e => setName(e.target.value)} maxLength={100} autoFocus required/>
                <TextField label="Description (optional)" value={description} onChange={e => setDescription(e.target.value)} maxLength={1000}/>
                {error && <p className="text-sm text-error" role="alert">{error}</p>}
                <div className="flex gap-2 justify-end">
                    <button type="button" className="btn btn-ghost" onClick={() => dialogRef.current?.close()}>Cancel</button>
                    <button type="submit" className="btn btn-primary" disabled={busy}>
                        {busy && <span className="loading loading-spinner loading-xs"/>}
                        {initial ? "Save" : "Create"}
                    </button>
                </div>
            </form>
            <form method="dialog" className="modal-backdrop"><button aria-label="Close">close</button></form>
        </dialog>,
        document.body
    );
}
