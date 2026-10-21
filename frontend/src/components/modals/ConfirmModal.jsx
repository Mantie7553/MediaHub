import { useState } from "react";
import { createPortal } from "react-dom";
import { Trash2 } from "lucide-react";

/**
 * Ask before doing something that can't be undone
 * @param {object} dialogRef ref for the <dialog>
 * @param {string} title the question, like Delete "Episode 3"?
 * @param {string[]} lines what will happen, shown as a list
 * @param {string} confirmLabel the button that goes ahead
 * @param {function} onConfirm runs the action; returns a promise, the dialog closes when it
 *  resolves and stays open (for a retry) when it rejects
 */
export default function ConfirmModal({ dialogRef, title, lines = [], confirmLabel = "Delete", onConfirm }) {
    const [busy, setBusy] = useState(false);

    function confirm() {
        setBusy(true);
        Promise.resolve(onConfirm?.())
            .then(() => dialogRef.current?.close())
            .catch(() => {})
            .finally(() => setBusy(false));
    }

    return createPortal(
        <dialog className="modal modal-bottom sm:modal-middle" ref={dialogRef}>
            <div className="modal-box flex flex-col gap-4">
                <h2 className="text-xl font-semibold flex items-center gap-2">
                    <Trash2 size={20} className="text-error shrink-0"/> <span className="min-w-0 break-words">{title}</span>
                </h2>
                {lines.length > 0 && <ul className="list-disc pl-6 text-sm flex flex-col gap-1 text-base-content/80">
                    {lines.map(line => <li key={line}>{line}</li>)}
                </ul>}
                <div className="flex gap-2 justify-end">
                    <button type="button" className="btn btn-ghost" onClick={() => dialogRef.current?.close()} disabled={busy}>Cancel</button>
                    <button type="button" className="btn btn-error" onClick={confirm} disabled={busy}>
                        {busy ? <span className="loading loading-spinner loading-xs"/> : <Trash2 size={16}/>} {confirmLabel}
                    </button>
                </div>
            </div>
            <form method="dialog" className="modal-backdrop"><button disabled={busy}>close</button></form>
        </dialog>,
        document.body
    );
}
