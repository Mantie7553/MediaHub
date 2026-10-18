import { Link } from "react-router-dom";
import Format from "../../utils/format";
import { mediaStatusBadge } from "../../utils/status";
import { useRef, useState } from "react";
import api from "../../services/api";
import AddToListModal from "../modals/AddToListModal";
import DeleteMediaModal from "../modals/DeleteMediaModal";
import { removeFromList } from "../../services/mediaService";
import { getRole } from "../../utils/auth";
import { Plus, Check } from "lucide-react";
import Cover from "../Cover";
import { toast } from "../../stores/useToastStore";
import useAudioStore from "../../stores/useAudioStore"

export default function Card({item, showActions=false, userContentMap={}, onListChange}) {
    let {infoSection, path} = mediaInfo(item);
    const userEntry = userContentMap[String(item.external_id ?? item.media_item_id ?? item.id)];
    const play = useAudioStore(state => state.play);
    // hidden as soon as an admin deletes it, without reloading the row it sits in
    const [deleted, setDeleted] = useState(false);

    if (deleted) return null;

    const card = (
        <li className="card border border-base-300 w-44 shrink-0 bg-base-300 h-full">
            <figure className="relative">
                {(userEntry?.status || item.status) && 
                    <span className={`badge ${mediaStatusBadge(userEntry?.status ?? item.status)} absolute top-2 left-2 z-10 text-xs p-1`}>
                        {Format.statusLabel(userEntry?.status ?? item.status, item.type ?? item.media_type)}
                    </span>}
                <Cover src={item.cover_image_url} source={item.external_source} alt={item.media_title ?? item.title} className="w-full h-56 object-contain"/>
            </figure>
            <div  className="card-body p-2 gap-1">
                <h3 className="card-title text-sm line-clamp-2">{item.media_title ?? item.title}</h3>
                {infoSection}
                <ActionButtons item={item} userEntry={userEntry} onListChange={onListChange} onDeleted={() => setDeleted(true)}/>
            </div>
        </li>
    )

    if (path === null) {
        return <div onClick={() => play({
            id: item.media_item_id || item.id,
            title: item.media_title || item.title,
            artist: item.artist,
            thumbnail: item.cover_image_url,
        })} className="cursor-pointer">{card}</div>
    }

    return item.media_item_id || item.id 
    ? <Link to={`${path}${item.media_item_id || item.id}`} state={{ from: window.location.pathname }}>
        {card}
        </Link>  
    : card
}

function mediaInfo(item) {
    let path = "";
    let info = null;
    switch(item.media_type ?? item.type) {
        case "anime": 
            const season = item.active_season ? `S${item.active_season}` : null
            const total = item.season_total
            const episodes = item.season_watched ?? null
            const progressLabel = episodes !== null 
                ? [season, `E${episodes}${total ? ` / ${total}` : ""}`].filter(Boolean).join(" · ")
                : null
            const progressPct = total ? Math.round((episodes / total) * 100) : 0

            info = <>
                    <span>{progressLabel}</span>
                    <progress className="progress progress-primary" value={progressPct} max="100"></progress>
                </>
            path = "/anime/"
            break
        case "light_novel": 
            path="/light-novels/"
            break
        case "movie":
            info = <>
                <span>{item.release_date ? Format.year(item.release_date) : null}</span>
            </>
            path = "/movies/"
            break
        case "manga":
            // info = <>
            //     <span>{`Chapter ${item.last_chapter}`}</span>
            // </>
            path = "/manga/"
            break
        case "music_track": 
            info = <>
                <span>{item.artist}</span>
            </>
            path = null
            break

        default: break
    }   

    return {infoSection: info, path: path}
}

function ActionButtons({item, userEntry, onListChange, onDeleted}) {
    const [loading, setLoading] = useState(false);
    const dialogRef = useRef(null);
    const deleteRef = useRef(null);
    // list entries carry media_item_id, library items their own id; search results that
    // aren't in the library have neither
    const mediaId = item.media_item_id ?? item.id;
    const canDelete = getRole() === "admin" && !!mediaId;

    function handleOpenModal(e) {
        e.stopPropagation();
        e.preventDefault();
        dialogRef.current.showModal();
    }

    function handleConfirm({ status, score, progress }) {
        setLoading(true);
        if (userEntry) {
            // existing list entry — update it
            api.put(`/me/media/${userEntry.id}`, {
                status,
                rating: score === 0 ? null : score,
            })
            .then(() => { toast.success("List updated"); onListChange?.(); })
            .catch(err => toast.error(err, "Could not update your list"))
            .finally(() => setLoading(false))
        } else if (item.id) {
            // already in DB — just add to list, no download
            api.post("/me/media", {
                media_item_id: item.id,
                status,
                rating: score === 0 ? null : score,
            })
            .then(() => { toast.success("Added to your list"); onListChange?.(); })
            .catch(err => toast.error(err, "Could not update your list"))
            .finally(() => setLoading(false))
        } else if (item.external_source) {
            // not in DB yet — save and dispatch download if permitted
            api.post("/search/save", {
                external_id: item.external_id,
                external_source: item.external_source,
                title: item.title,
                cover_image_url: item.cover_image_url,
                type: item.type,
                action: status === "wishlist" ? "both" : "list",
                status,
                rating: score === 0 ? null : score,
                progress,
            })
            .then(() => { toast.success("Added to your list"); onListChange?.(); })
            .catch(err => toast.error(err, "Could not update your list"))
            .finally(() => setLoading(false))
        }
    }

    function handleRemove() {
        setLoading(true);
        removeFromList(userEntry.id)
            .then(() => { toast.success("Removed from your list"); onListChange?.(); })
            .catch(err => toast.error(err, "Could not remove it from your list"))
            .finally(() => setLoading(false))
    }

    const isLibraryItem = !!(userEntry || item.id);

    return <>
        <AddToListModal item={item} onConfirm={handleConfirm} dialogRef={dialogRef} initialValues={userEntry} isLibraryItem={isLibraryItem}
            onRemove={userEntry ? handleRemove : undefined}
            onDelete={canDelete ? () => deleteRef.current.showModal() : undefined}/>
        {canDelete && <DeleteMediaModal
            dialogRef={deleteRef}
            item={{ id: mediaId, title: item.media_title ?? item.title, type: item.type ?? item.media_type }}
            onDeleted={() => { onDeleted?.(); onListChange?.(); }}
        />}
        <div className="flex flex-col gap-1">
            <button className="absolute top-2 right-2 z-10 btn btn-circle btn-xs btn-primary text-lg" onClick={handleOpenModal} disabled={loading}>
                {userEntry ? <Check size={14} strokeWidth={4}/> : <Plus size={14} strokeWidth={4}/>}
            </button>
        </div>
    </>
}