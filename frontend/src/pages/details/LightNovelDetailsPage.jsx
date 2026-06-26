import { useState, useEffect } from "react";
import { NavLink, useLocation, useParams } from "react-router-dom";
import Loading from "../../components/states/Loading";
import Error from "../../components/states/Error";
import { useMediaItem, useUserContent } from "../../hooks";
import { mediaStatusBadge } from "../../utils/status";
import api from "../../services/api";
import { ArrowLeft, Check } from "lucide-react";
import Format from "../../utils/format";

export default function LightNovelDetailsPage() {
    const navigate = useNavigate();
    const { userContentMap, refresh } = useUserContent();
    const { id } = useParams();
    const { item: ln, loading, error } = useMediaItem(id);
    const userEntry = userContentMap[id];
    const [readVolumes, setReadVolumes] = useState(new Set());
    const [scrollPosition, setScrollPosition] = useState({});

    useEffect(() => {
        if (ln?.metadata?.volumes) {
            setReadVolumes(new Set(ln.metadata.volumes.filter(v => v.completed).map(v => v.id)));
            const initial = {};
            ln.metadata.volumes.forEach(v => { initial[v.id] = v.scroll_position ?? 0; });
            setScrollPosition(initial);
        }
    }, [ln]);

    function updateStatus(status) {
        if (userEntry) {
            api.put(`/me/media/${userEntry.id}`, { status }).then(() => refresh());
        } else {
            api.post(`/me/media`, { media_item_id: id, status }).then(() => refresh());
        }
    }

    function toggleVolume(volume) {
        const read = !readVolumes.has(volume.id);
        api.put(`/light-novels/volumes/${volume.id}/read`, { read })
            .then(() => {
                const next = new Set(readVolumes);
                read ? next.add(volume.id) : next.delete(volume.id);
                const status = next.size === ln.metadata.volumes.length ? "completed" : "current";
                setReadVolumes(next);
                updateStatus(status);
            })
            .catch(() => {});
        api.put(`/light-novels/volumes/${volume.id}/progress`, { scroll_position: read ? 1 : 0 })
            .then(() => setScrollPosition(prev => ({ ...prev, [volume.id]: read ? 1 : 0 })));
    }

    function markAll(read) {
        api.put(`/light-novels/${id}/read`, { read })
            .then(() => {
                setReadVolumes(read ? new Set(ln.metadata.volumes.map(v => v.id)) : new Set());
                updateStatus(read ? "completed" : "planned");
                const newScrollPositions = {};
                ln.metadata.volumes.forEach(v => { newScrollPositions[v.id] = read ? 1 : 0; });
                setScrollPosition(newScrollPositions);
                Promise.all(ln.metadata.volumes.map(v =>
                    api.put(`/light-novels/volumes/${v.id}/progress`, { scroll_position: read ? 1 : 0 })
                )).catch(() => {});
            })
            .catch(() => {});
    }

    if (loading) return <Loading />
    if (error) return <Error error={error} />
    if (!ln) return null

    return (
        <div className="flex flex-col">
            <button className="btn btn-ghost btn-sm self-start" onClick={() => navigate("/")}>
                <ArrowLeft size={18} strokeWidth={2}/> Back
            </button>
            <div className="flex gap-6">
                <img src={ln.cover_image_url} className="w-48 h-64 object-contain rounded-md" />
                <div className="flex flex-col gap-3">
                    <h2 className="text-2xl font-bold">{ln.title}</h2>
                    {ln.metadata.author && <span className="text-sm text-neutral-content">by {ln.metadata.author}</span>}
                    {ln.release_date && (
                        <span className="text-sm text-neutral-content">{Format.year(ln.release_date)}</span>
                    )}
                    <div className="flex flex-wrap gap-1">
                        {(ln.metadata.genres ?? []).map((genre, i) => (
                            <span key={i} className="badge">{genre}</span>
                        ))}
                    </div>
                    {ln.description && (
                        <p className="text-sm max-w-xl" dangerouslySetInnerHTML={{ __html: ln.description }} />
                    )}
                    <div className="flex flex-wrap items-center gap-2">
                        <div className="dropdown">
                            <div tabIndex={0} className={`badge ${mediaStatusBadge(userEntry?.status)} cursor-pointer`}>
                                {Format.statusLabel(userEntry?.status, ln.type)}
                            </div>
                            <ul tabIndex={0} className="dropdown-content menu bg-base-200 rounded-box z-10 p-2 shadow gap-1">
                                {["current", "completed", "dropped", "planned"].map(option => (
                                    <li key={option}>
                                        <button onClick={() => { updateStatus(option); document.activeElement.blur(); }}>
                                            {Format.statusLabel(option, ln.type)}
                                        </button>
                                    </li>
                                ))}
                            </ul>
                        </div>
                        <button className="btn btn-outline btn-sm" onClick={() => markAll(true)}>Mark all read</button>
                        <button className="btn btn-outline btn-sm" onClick={() => markAll(false)}>Mark all unread</button>
                    </div>
                </div>
            </div>

            <h3 className="font-bold text-lg mt-6">Volumes</h3>
            <div className="flex flex-col gap-2 mt-2">
                {(ln.metadata.volumes ?? []).map(volume => (
                    <div key={volume.id} className="flex items-center gap-3 p-3 rounded-lg bg-base-200">
                        <button
                            className={`btn btn-circle btn-xs ${readVolumes.has(volume.id) ? "btn-primary" : "btn-outline"}`}
                            onClick={() => toggleVolume(volume)}
                        >
                            <Check size={10} strokeWidth={3} />
                        </button>
                        <NavLink to={`/light-novels/${id}/volumes/${volume.id}/read`} className="text-sm w-24 shrink-0">
                            {volume.title ?? `Volume ${volume.volume_number}`}
                        </NavLink>
                        {scrollPosition[volume.id] != null && (
                            <div className="relative flex-1 max-w-48">
                                <progress
                                    className="progress progress-primary w-full"
                                    value={Math.round((scrollPosition[volume.id] ?? 0) * 100)}
                                    max="100"
                                />
                                <span className="absolute inset-0 flex items-center justify-center text-xs font-bold">
                                    {Math.round((scrollPosition[volume.id] ?? 0) * 100)}%
                                </span>
                            </div>
                        )}
                    </div>
                ))}
            </div>
        </div>
    )
}