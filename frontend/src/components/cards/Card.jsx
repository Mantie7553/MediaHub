import { Link } from "react-router-dom";
import Format from "../../utils/format";
import { mediaStatusBadge } from "../../utils/status";
import { useRef, useState } from "react";
import api from "../../services/api";
import AddToListModal from "../modals/AddToListModal";
import { Plus, Check } from "lucide-react";

export default function Card({item, showActions=false, userContentMap={}, onListChange}) {
    let {infoSection, path} = mediaInfo(item);
    const userEntry = userContentMap[String(item.external_id ?? item.media_item_id ?? item.id)];

    const card = (
        <li className="card border border-base-300 w-44 shrink-0 bg-base-300 h-full">
            <figure className="relative">
                {(userEntry?.status || item.status) && 
                    <span className={`badge ${mediaStatusBadge(userEntry?.status ?? item.status)} absolute top-2 left-2 z-10 text-xs p-1`}>
                        {Format.cleanString(userEntry?.status ?? item.status)}
                    </span>}
                {item.cover_image_url ? (
                    <img src={item.cover_image_url} className="w-full h-56 object-contain"/>
                ) : (
                    <div className="skeleton h-64 w-full"></div>
                )}
            </figure>
            <div  className="card-body p-2 gap-1">
                <h3 className="card-title text-sm line-clamp-2">{item.media_title ?? item.title}</h3>
                {infoSection}
                <ActionButtons item={item} userEntry={userEntry} onListChange={onListChange}/>
            </div>
        </li>
    )

    return <Link to={`${path}${item.media_item_id || item.id}`}>{card}</Link>
}

function mediaInfo(item) {
    let path = "";
    let info = null;
    switch(item.media_type ?? item.type) {
        case "anime": 
            const season = item.season_number ? `S${item.season_number}` : null
            const total = item.episode_count
            const episodes = item.episodes_watched ?? null
            const progressLabel = episodes !== null 
                ? [season, `E${episodes}${total ? ` / ${total}` : ""}`].filter(Boolean).join(" · ")
                : null
            const progressPct = total ? Math.round((episodes / total) * 100) : 0

            info = <>
                    <span>{progressLabel}</span>
                    <progress className="progress" value={progressPct} max="100"></progress>
                </>
            path = "/anime/"
            break
        case "movie":
            info = <>
                <span>{item.release_date ? Format.year(item.release_date) : null}</span>
            </>
            path = "/watch/movie/"
            break
        case "manga":
            // info = <>
            //     <span>{`Chapter ${item.last_chapter}`}</span>
            // </>
            path = "/manga/"
            break
        case "music": 
            info = <>
                <span>{item.artist}</span>
            </>
            break

        default: break
    }   

    return {infoSection: info, path: path}
}

function ActionButtons({item, userEntry, onListChange}) {
    const [msg, setMsg] = useState("");
    const [loading, setLoading] = useState(false);
    const dialogRef = useRef(null);

    function handleOpenModal(e) {
        e.stopPropagation();
        e.preventDefault();
        dialogRef.current.showModal();
    }

    function handleConfirm({ status, score, progress }) {
        setLoading(true);
        setMsg("");
        if (userEntry) {
            // existing list entry — update it
            api.put(`/me/media/${userEntry.id}`, {
                status,
                rating: score === 0 ? null : score,
            })
            .then(() => { setMsg("Updated!"); onListChange?.(); })
            .catch(err => setMsg(err.response?.data?.error ?? err.message))
            .finally(() => setLoading(false))
        } else if (item.external_source) {
            // search result — needs full save with external metadata
            api.post("/search/save", {
                external_id: item.external_id,
                external_source: item.external_source,
                title: item.title,
                cover_image_url: item.cover_image_url,
                type: item.type,
                action: "both",
                status,
                rating: score === 0 ? null : score,
                progress,
            })
            .then(() => { setMsg("Added to list!"); onListChange?.(); })
            .catch(err => setMsg(err.response?.data?.error ?? err.message))
            .finally(() => setLoading(false))
        } else {
            // library item — already in DB, just add to list
            api.post("/me/media", {
                media_item_id: item.id,
                status,
                rating: score === 0 ? null : score,
            })
            .then(() => { setMsg("Added to list!"); onListChange?.(); })
            .catch(err => setMsg(err.response?.data?.error ?? err.message))
            .finally(() => setLoading(false))
        }
    }

    return <>
        <AddToListModal item={item} onConfirm={handleConfirm} dialogRef={dialogRef} initialValues={userEntry}/>
        <div className="flex flex-col gap-1">
            <button className="absolute top-2 right-2 z-10 btn btn-circle btn-xs btn-primary text-lg" onClick={handleOpenModal} disabled={loading}>
                {userEntry ? <Check size={14} strokeWidth={4}/> : <Plus size={14} strokeWidth={4}/>}
            </button>
        </div>
        {msg && <p className="text-xs mt-1">{msg}</p>}
    </>
}