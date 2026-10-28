import { useRef } from "react";
import { createPortal } from "react-dom";
import { useNavigate } from "react-router-dom";
import { Trash2 } from "lucide-react";
import DeleteMediaModal from "./modals/DeleteMediaModal";
import { userIsAdmin } from "../utils/auth";

/**
 * Admin only: a button on a details page that permanently deletes the title, then leaves the
 * page (replacing it in history, so Back can't return to something that no longer exists)
 * @param {object} item { id, title, type }, type "music_album" for an album
 * @param {boolean} asMenuItem renders an item for an ActionMenu instead of a button
 */
export default function DeleteFromServerButton({ item, asMenuItem = false }) {
    const navigate = useNavigate();
    const dialogRef = useRef(null);

    if (!userIsAdmin()) return null;

    function open() {
        document.activeElement?.blur();
        dialogRef.current.showModal();
    }

    // the dialog is drawn on <body>: inside a closed dropdown menu it would inherit the menu's
    // hidden visibility and open invisibly
    const modal = createPortal(
        <DeleteMediaModal dialogRef={dialogRef} item={item} onDeleted={() => navigate("/", { replace: true })}/>,
        document.body
    );

    if (asMenuItem) return <li className="border-t border-white/5 mt-1 pt-1">
        <button className="text-error" onClick={open}><Trash2 size={16}/> Delete from server…</button>
        {modal}
    </li>

    return <>
        <button className="btn btn-outline btn-error btn-sm w-fit" onClick={open}>
            <Trash2 size={16}/> Delete from server
        </button>
        {modal}
    </>
}
