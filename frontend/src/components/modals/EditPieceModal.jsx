import { useState } from "react";
import { createPortal } from "react-dom";
import { PencilLine } from "lucide-react";
import TextField from "../TextField";
import { toast } from "../../stores/useToastStore";
import { editMedia, editPiece } from "../../services/curationService";
import { changedFields } from "../../utils/curation";

// the fields each kind of piece can change, and how they are sent
const kinds = {
    episode: {
        noun: "episode",
        fields: [{ key: "title", label: "Title", hint: "Leave empty to use Sonarr's title again" }],
    },
    chapter: {
        noun: "chapter",
        fields: [
            { key: "chapter_number", label: "Chapter number", inputMode: "decimal", required: true },
            { key: "title", label: "Title", hint: "Leave empty to show it as \"Chapter N\"" },
        ],
    },
    volume: {
        noun: "volume",
        fields: [
            { key: "volume_number", label: "Volume number", number: true, required: true },
            { key: "title", label: "Title", hint: "Leave empty to show it as \"Volume N\"" },
        ],
    },
    track: {
        noun: "track",
        fields: [
            { key: "title", label: "Title", required: true },
            { key: "artist", label: "Artist", required: true },
            { key: "track_number", label: "Track number", number: true },
        ],
    },
};

/**
 * Admin: rename or renumber one episode, chapter, volume or track
 * @param {object} dialogRef ref for the <dialog>
 * @param {object} piece { kind, id, label, ...the current values of its fields }
 * @param {function} onSaved called after saving
 */
export default function EditPieceModal({ dialogRef, piece, onSaved }) {
    return createPortal(
        <dialog className="modal modal-bottom sm:modal-middle" ref={dialogRef}>
            {/* keyed by the piece, so each opening starts from its saved values */}
            {piece && <PieceForm key={`${piece.kind}-${piece.id}-${piece.version ?? 0}`} dialogRef={dialogRef} piece={piece} onSaved={onSaved}/>}
            <form method="dialog" className="modal-backdrop"><button>close</button></form>
        </dialog>,
        document.body
    );
}

function PieceForm({ dialogRef, piece, onSaved }) {
    const kind = kinds[piece.kind];
    const [initial] = useState(() => Object.fromEntries(kind.fields.map(f => [f.key, piece[f.key] == null ? "" : String(piece[f.key])])));
    const [values, setValues] = useState(initial);
    const [saving, setSaving] = useState(false);

    function save(e) {
        e.preventDefault();
        const changes = changedFields(initial, values, kind.fields.filter(f => f.number).map(f => f.key));
        if (Object.keys(changes).length === 0) {
            dialogRef.current.close();
            return;
        }
        setSaving(true);
        const request = piece.kind === "track" ? editMedia(piece.id, changes) : editPiece(piece.kind, piece.id, changes);
        request
            .then(() => {
                toast.success("Saved");
                dialogRef.current.close();
                onSaved?.();
            })
            .catch(err => toast.error(err, `Could not save the ${kind.noun}`))
            .finally(() => setSaving(false));
    }

    return <form className="modal-box flex flex-col gap-4" onSubmit={save}>
        <div className="flex items-center gap-3">
            <span className="grid size-10 place-items-center rounded-full bg-primary/15 text-primary"><PencilLine size={18}/></span>
            <div className="min-w-0">
                <h3 className="text-xl font-semibold capitalize">Edit {kind.noun}</h3>
                <p className="text-sm text-base-content/50 truncate">{piece.label}</p>
            </div>
        </div>
        {kind.fields.map(field => <TextField
            key={field.key}
            label={field.label}
            hint={field.hint}
            value={values[field.key]}
            onChange={e => setValues(v => ({ ...v, [field.key]: e.target.value }))}
            required={field.required}
            maxLength={300}
            {...(field.number ? { type: "number", min: 0, inputMode: "numeric" } : {})}
            {...(field.inputMode ? { inputMode: field.inputMode, pattern: "\\d+(\\.\\d+)?" } : {})}
        />)}
        {piece.kind === "chapter" && values.chapter_number !== initial.chapter_number && <p className="text-xs text-base-content/50">
            The chapter's file is renamed to match its new number.
        </p>}
        <div className="flex justify-end gap-2 pt-1">
            <button type="button" className="btn btn-ghost" onClick={() => dialogRef.current.close()} disabled={saving}>Cancel</button>
            <button type="submit" className="btn btn-primary" disabled={saving}>
                {saving && <span className="loading loading-spinner loading-xs"/>} Save
            </button>
        </div>
    </form>
}
