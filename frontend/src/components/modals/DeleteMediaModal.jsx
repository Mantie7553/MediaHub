import { useState } from "react";
import { Trash2 } from "lucide-react";
import { deleteAlbum, deleteMedia } from "../../services/mediaService";
import { toast } from "../../stores/useToastStore";

/**
 * Admin confirmation for permanently deleting a title. The Delete button unlocks only once the
 * title has been typed, since this removes files and can't be undone
 * @param {object} dialogRef ref for the <dialog>, open it with dialogRef.current.showModal()
 * @param {object} item { id, title, type } of the media item, or of the album with type "music_album"
 * @param {function} onDeleted called after the server deleted it
 */
export default function DeleteMediaModal({ dialogRef, item, onDeleted }) {
    const [typed, setTyped] = useState("");
    const [deleting, setDeleting] = useState(false);

    const title = item?.title ?? "";
    const type = item?.type;
    const matches = typed.trim().toLowerCase() === title.trim().toLowerCase() && title !== "";

    const consequences = [
        "Deletes its files from the NAS",
        ...(type === "anime" ? ["Removes the series from Sonarr"] : []),
        ...(type === "movie" ? ["Removes the movie from Radarr"] : []),
        ...(type === "music_album" ? ["Deletes every track on the album"] : []),
        "Removes it from everyone's lists and progress",
        "This can't be undone",
    ];

    function close() {
        dialogRef.current?.close();
    }

    function handleDelete() {
        if (!matches || deleting) return;
        setDeleting(true);
        const request = type === "music_album" ? deleteAlbum(item.id) : deleteMedia(item.id);
        request
            .then(() => {
                toast.success(`Deleted "${title}"`);
                close();
                onDeleted?.();
            })
            .catch(err => toast.error(err, `Could not delete "${title}"`))
            .finally(() => setDeleting(false));
    }

    return <dialog
        className="modal modal-bottom sm:modal-middle"
        ref={dialogRef}
        // cards render this inside their link, keep clicks inside from following it
        onClick={e => { e.stopPropagation(); e.preventDefault(); }}
        onClose={() => setTyped("")}
    >
        <div className="modal-box flex flex-col gap-4">
            <h2 className="text-xl font-semibold flex items-center gap-2">
                <Trash2 size={20} className="text-error"/> Delete "{title}"?
            </h2>
            <ul className="list-disc pl-6 text-sm flex flex-col gap-1">
                {consequences.map(c => <li key={c}>{c}</li>)}
            </ul>
            <label className="flex flex-col gap-2">
                <span className="text-sm">Type <strong className="select-all">{title}</strong> to confirm</span>
                <input
                    className="input input-bordered w-full"
                    value={typed}
                    onChange={e => setTyped(e.target.value)}
                    onKeyDown={e => e.key === "Enter" && handleDelete()}
                    autoComplete="off"
                    aria-label="Title to confirm"
                />
            </label>
            <div className="flex gap-2 justify-end">
                <button type="button" className="btn btn-ghost" onClick={close} disabled={deleting}>Cancel</button>
                <button type="button" className="btn btn-error" onClick={handleDelete} disabled={!matches || deleting}>
                    {deleting ? <span className="loading loading-spinner loading-xs"/> : <Trash2 size={16}/>}
                    Delete permanently
                </button>
            </div>
        </div>
    </dialog>
}
