import { Link } from "react-router-dom";
import Format from "../../utils/format";
import { mediaStatusDot } from "../../utils/status";
import { useRef, useState } from "react";
import api from "../../services/api";
import AddToListModal from "../modals/AddToListModal";
import DeleteMediaModal from "../modals/DeleteMediaModal";
import { removeFromList } from "../../services/mediaService";
import { getRole } from "../../utils/auth";
import { Plus, Check, Play } from "lucide-react";
import Cover from "../Cover";
import { toast } from "../../stores/useToastStore";
import useAudioStore from "../../stores/useAudioStore"

/**
 * A poster card for a title, linking to its details page (tracks play instead)
 * @param {boolean} fluid fills its grid cell instead of the fixed width used in rows
 */
export default function Card({item, showActions=false, userContentMap={}, onListChange, fluid=false}) {
    let {meta, progressPct, path} = mediaInfo(item);
    const userEntry = userContentMap[String(item.external_id ?? item.media_item_id ?? item.id)];
    const play = useAudioStore(state => state.play);
    // hidden as soon as an admin deletes it, without reloading the row it sits in
    const [deleted, setDeleted] = useState(false);

    if (deleted) return null;

    const status = userEntry?.status ?? item.status;
    const title = item.media_title ?? item.title;
    const isTrack = path === null;

    const card = <>
        <div className="relative aspect-[2/3] overflow-hidden rounded-box bg-base-300 shadow-card ring-1 ring-white/5
            transition-all duration-300 ease-out
            group-hover:-translate-y-1 group-hover:shadow-raised group-hover:ring-primary/40
            group-focus-within:ring-primary/60">
            <Cover src={item.cover_image_url} source={item.external_source} alt={title}
                className="size-full object-cover transition-transform duration-500 ease-out group-hover:scale-[1.04]"/>
            {/* darkens the bottom on hover so the controls over it stay readable */}
            <div className="absolute inset-x-0 bottom-0 h-2/5 bg-linear-to-t from-black/70 to-transparent opacity-0 transition-opacity duration-300 group-hover:opacity-100 pointer-events-none"></div>
            {status &&
                <span className="glass absolute top-2 left-2 z-10 flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[11px] font-medium text-white">
                    <span className={`size-1.5 rounded-full ${mediaStatusDot(status)}`}></span>
                    {Format.statusLabel(status, item.type ?? item.media_type)}
                </span>}
            {isTrack && <span className="absolute bottom-3 right-3 z-10 grid size-10 place-items-center rounded-full bg-primary text-primary-content shadow-glow
                opacity-0 translate-y-2 transition-all duration-300 group-hover:opacity-100 group-hover:translate-y-0 hover-reveal">
                <Play size={18} fill="currentColor"/>
            </span>}
            <ActionButtons item={item} userEntry={userEntry} onListChange={onListChange} onDeleted={() => setDeleted(true)}/>
            {progressPct > 0 && <div className="absolute inset-x-0 bottom-0 h-1 bg-black/50">
                <div className="h-full bg-primary shadow-[0_0_8px_rgb(242_89_18/0.8)]" style={{ width: `${progressPct}%` }}></div>
            </div>}
        </div>
        <div className="mt-2.5 px-0.5">
            <h3 className="text-sm font-medium leading-snug line-clamp-2 transition-colors group-hover:text-primary">{title}</h3>
            {meta && <p className="mt-0.5 text-xs text-base-content/50 truncate">{meta}</p>}
        </div>
    </>

    const id = item.media_item_id || item.id;
    let body;
    if (isTrack) {
        const playTrack = () => play({
            id,
            title,
            artist: item.artist,
            thumbnail: item.cover_image_url,
        });
        body = <div role="button" tabIndex={0} className="block cursor-pointer outline-none" onClick={playTrack}
            onKeyDown={e => e.key === "Enter" && e.target === e.currentTarget && playTrack()}>{card}</div>
    } else if (id) {
        body = <Link to={`${path}${id}`} state={{ from: window.location.pathname }} className="block outline-none">
            {card}
        </Link>
    } else {
        body = card
    }

    return <li className={`group ${fluid ? "min-w-0" : "w-[8.5rem] sm:w-40 md:w-44 shrink-0"}`}>{body}</li>
}

/**
 * What a card shows under its title, and where it links to
 * @returns meta (the line under the title), progressPct (0 hides the bar) and path (null for tracks)
 */
function mediaInfo(item) {
    let path = "";
    let meta = null;
    let progressPct = 0;
    switch(item.media_type ?? item.type) {
        case "anime": {
            const season = item.active_season ? `S${item.active_season}` : null
            const total = item.season_total
            const episodes = item.season_watched ?? null
            meta = episodes !== null
                ? [season, `E${episodes}${total ? ` / ${total}` : ""}`].filter(Boolean).join(" · ")
                : (item.release_date ? Format.year(item.release_date) : null)
            progressPct = total ? Math.round((episodes / total) * 100) : 0
            path = "/anime/"
            break
        }
        case "light_novel":
            path="/light-novels/"
            break
        case "movie":
            meta = item.release_date ? Format.year(item.release_date) : null
            path = "/movies/"
            break
        case "manga":
            path = "/manga/"
            break
        case "music_track":
            meta = item.artist
            path = null
            break

        default: break
    }

    return {meta, progressPct, path}
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
        <button
            className={`absolute top-2 right-2 z-10 grid size-8 place-items-center rounded-full transition-all duration-200
                hover:scale-110 disabled:opacity-50
                ${userEntry
                    ? "bg-primary text-primary-content shadow-glow"
                    : "glass text-white opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 focus-visible:opacity-100 hover:bg-primary hover-reveal"}`}
            onClick={handleOpenModal}
            disabled={loading}
            aria-label={userEntry ? "Edit list entry" : "Add to list"}
        >
            {loading
                ? <span className="loading loading-spinner loading-xs"/>
                : userEntry ? <Check size={15} strokeWidth={3}/> : <Plus size={16} strokeWidth={2.75}/>}
        </button>
    </>
}
