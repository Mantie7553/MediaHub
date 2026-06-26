import { useState, useEffect } from "react";
import { NavLink, useParams } from "react-router-dom";
import { mangaBadge, mediaStatusBadge } from "../../utils/status";
import Loading from "../../components/states/Loading";
import Error from "../../components/states/Error";
import { useMediaItem, useUserContent } from "../../hooks";
import api from "../../services/api";
import { Check } from "lucide-react";
import Format from "../../utils/format";

export default function MangaDetailsPage() {
    const { userContentMap, refresh } = useUserContent();
    const { id } = useParams();
    const { item: manga, loading, error } = useMediaItem(id);
    const userEntry = userContentMap[id];
    const [readChapters, setReadChapters] = useState(new Set());

    useEffect(() => {
        if (manga?.metadata?.chapters) {
            setReadChapters(new Set(manga.metadata.chapters.filter(c => c.completed).map(c => c.id)));
        }
    }, [manga]);

    function updateStatus(status) {
        if (userEntry) {
            api.put(`/me/media/${userEntry.id}`, { status }).then(() => refresh());
        } else {
            api.post(`/me/media`, { media_item_id: id, status }).then(() => refresh());
        }
    }

    function toggleChapter(chapter) {
        const read = !readChapters.has(chapter.id);
        api.put(`/manga/chapters/${chapter.id}/read`, { read })
            .then(() => {
                const next = new Set(readChapters);
                read ? next.add(chapter.id) : next.delete(chapter.id);
                const status = next.size === manga.metadata.chapters.length ? "completed" : "manga_reading";
                setReadChapters(next);
                updateStatus(status);
            })
            .catch(() => {});
    }

    function markAll(read) {
        api.put(`/manga/${id}/read`, { read })
            .then(() => {
                setReadChapters(read ? new Set(manga.metadata.chapters.map(c => c.id)) : new Set());
                updateStatus(read ? "completed" : "plan_to_watch");
            })
            .catch(() => {});
    }

    if (loading) return <Loading />
    if (error) return <Error error={error} />
    if (!manga) return null

    return (
        <div className="flex flex-col">
            <div className="flex gap-6">
                <img src={manga.cover_image_url} className="w-48 h-64 object-cover rounded-md" />
                <div className="flex flex-col gap-3">
                    <h2 className="text-2xl font-bold">{manga.title}</h2>
                    <span className={`badge ${mangaBadge(manga.metadata.status)}`}>{manga.metadata.status}</span>
                    <span className="text-sm text-neutral-content">{manga.metadata.total_chapters ?? "N/A"} chapters</span>
                    <div className="flex flex-wrap gap-1">
                        {(manga.metadata.genres ?? []).map((genre, i) => (
                            <span key={`${manga.title}-${i}`} className="badge">{genre}</span>
                        ))}
                    </div>
                    {manga.description && (
                        <p className="text-sm max-w-xl">{manga.description}</p>
                    )}
                    <div className="flex flex-wrap items-center gap-2">
                        <div className="dropdown">
                            <div tabIndex={0} className={`badge ${mediaStatusBadge(userEntry?.status)} cursor-pointer`}>
                                {Format.cleanString(userEntry?.status ?? "Add to list")}
                            </div>
                            <ul tabIndex={0} className="dropdown-content menu bg-base-200 rounded-box z-10 p-2 shadow gap-1">
                                {["manga_reading", "completed", "dropped", "plan_to_watch"].map(option => (
                                    <li key={option}>
                                        <button onClick={() => { updateStatus(option); document.activeElement.blur(); }}>
                                            {Format.cleanString(option)}
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

            {manga.metadata.chapters && <>
                <h3 className="font-bold text-lg mt-4">Chapters</h3>
                <div className="flex flex-col gap-2 mt-2">
                    {manga.metadata.chapters.map(chapter => (
                        <div key={chapter.id} className="flex items-center justify-between p-3 rounded-lg bg-base-200">
                            <div className="flex items-center gap-3">
                                <button
                                    className={`btn btn-circle btn-xs ${readChapters.has(chapter.id) ? "btn-primary" : "btn-outline"}`}
                                    onClick={() => toggleChapter(chapter)}
                                >
                                    <Check size={10} strokeWidth={3} />
                                </button>
                                <NavLink to={`/manga/${id}/chapters/${chapter.id}/read`} className="text-sm">
                                    {chapter.title ?? `Chapter ${chapter.chapter_number}`}
                                </NavLink>
                            </div>
                        </div>
                    ))}
                </div>
            </>}
        </div>
    )
}