import { useRef } from "react";
import { useNavigate } from "react-router-dom";
import { Trash2 } from "lucide-react";
import DeleteMediaModal from "./modals/DeleteMediaModal";
import { getRole } from "../utils/auth";

/**
 * Admin only: a button on a details page that permanently deletes the title, then leaves the
 * page (replacing it in history, so Back can't return to something that no longer exists)
 * @param {object} item { id, title, type }, type "music_album" for an album
 */
export default function DeleteFromServerButton({ item }) {
    const navigate = useNavigate();
    const dialogRef = useRef(null);

    if (getRole() !== "admin") return null;

    return <>
        <button className="btn btn-outline btn-error btn-sm w-fit" onClick={() => dialogRef.current.showModal()}>
            <Trash2 size={16}/> Delete from server
        </button>
        <DeleteMediaModal dialogRef={dialogRef} item={item} onDeleted={() => navigate("/", { replace: true })}/>
    </>
}
