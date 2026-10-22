import { useRef, useState } from "react";
import { PencilLine, Trash2 } from "lucide-react";
import { ActionMenu, ActionMenuItem } from "../components/details/Menus";
import EditPieceModal from "../components/modals/EditPieceModal";
import ConfirmModal from "../components/modals/ConfirmModal";
import { deletePiece } from "../services/curationService";
import { deleteMedia } from "../services/mediaService";
import { toast } from "../stores/useToastStore";
import { getRole } from "../utils/auth";

// what deleting each kind of piece does, shown before it happens
const consequences = {
    episode: [
        "Deletes the episode's file from the NAS",
        "Sonarr stops monitoring it, so it isn't downloaded again",
        "Removes everyone's progress for it",
    ],
    chapter: [
        "Deletes the chapter's file from the NAS",
        "Removes everyone's progress for it",
        "Checking MangaDex for new chapters downloads it again if MangaDex has it",
    ],
    volume: [
        "Deletes the volume's file from the NAS",
        "Removes everyone's progress for it",
    ],
    track: [
        "Deletes the track's file from the NAS",
        "Removes it from everyone's lists",
    ],
};

/**
 * The ⋯ menu on each episode, chapter, volume or track row and the dialogs it opens. Admins get
 * edit and delete; anyone gets the extra items a page passes in, like "Add to playlist". With
 * neither, menu() is null and modals is empty
 * @param {function} onChanged called after a piece was edited or deleted, to reload the list
 * @returns menu(piece, extra) for a row, where piece is { kind, id, label, ...its editable
 *  values } and extra is ActionMenuItem elements shown above the admin ones,
 *  and modals to render once on the page
 */
export default function useAdminPieces(onChanged) {
    const editRef = useRef(null);
    const deleteRef = useRef(null);
    const [piece, setPiece] = useState(null);
    const isAdmin = getRole() === "admin";

    function open(ref, p) {
        // a new version each opening, so the edit form starts from the row's current values
        setPiece({ ...p, version: Date.now() });
        // after the dialog has the piece
        setTimeout(() => ref.current?.showModal());
    }

    function remove() {
        const request = piece.kind === "track" ? deleteMedia(piece.id) : deletePiece(piece.kind, piece.id);
        return request
            .then(() => {
                toast.success(`Deleted ${piece.label}`);
                onChanged?.();
            })
            .catch(err => {
                toast.error(err, `Could not delete ${piece.label}`);
                throw err;
            });
    }

    const menu = (p, extra) => (isAdmin || extra) && <ActionMenu label={`${p.label} actions`} size="sm">
        {extra}
        {isAdmin && <>
            <ActionMenuItem icon={PencilLine} label="Edit…" onClick={() => open(editRef, p)}/>
            <ActionMenuItem icon={Trash2} label="Delete…" danger onClick={() => open(deleteRef, p)}/>
        </>}
    </ActionMenu>;

    const modals = isAdmin && <>
        <EditPieceModal dialogRef={editRef} piece={piece} onSaved={onChanged}/>
        <ConfirmModal
            dialogRef={deleteRef}
            title={piece ? `Delete ${piece.label}?` : ""}
            lines={piece ? consequences[piece.kind] : []}
            confirmLabel="Delete"
            onConfirm={remove}
        />
    </>;

    return { menu, modals };
}
