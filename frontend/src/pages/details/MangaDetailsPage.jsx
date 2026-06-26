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
    const [pageProgress, setPageProgress] = useState({});

    useEffect(() => {
        if (manga?.metadata?.chapters) {
            const initial = {};
            manga.metadata.chapters.forEach(c => { initial[c.id] = c.last_page_read ?? 0; });
            setPageProgress(initial);
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
                const status = next.size === manga.metadata.chapters.length ? "completed" : "current";
                setReadChapters(next);
                updateStatus(status);
            })
            .catch(() => {});
        api.put(`/manga/${id}/chapters/${chapter.id}/progress`, 
            { last_page_read: read ? chapter.page_count -1 : 0, completed: read })
        .then(() => setPageProgress(prev => ({ ...prev, [chapter.id]: read ? (chapter.page_count - 1) : 0 })));
    }

    function markAll(read) {
        api.put(`/manga/${id}/read`, { read })
            .then(() => {
                setReadChapters(read ? new Set(manga.metadata.chapters.map(c => c.id)) : new Set());
                updateStatus(read ? "completed" : "planned");
                const newPageProgress = {};
                manga.metadata.chapters.forEach(c => { newPageProgress[c.id] = read ? (c.page_count - 1) : 0; });
                setPageProgress(newPageProgress);
                Promise.all(manga.metadata.chapters.map(c =>
                    api.put(`/manga/${id}/chapters/${c.id}/progress`, { last_page_read: read ? (c.page_count - 1) : 0, completed: read })
                )).catch(() => {});
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
                                {Format.statusLabel(userEntry?.status, manga.type)}
                            </div>
                            <ul tabIndex={0} className="dropdown-content menu bg-base-200 rounded-box z-10 p-2 shadow gap-1">
                                {["current", "completed", "dropped", "planned"].map(option => (
                                    <li key={option}>
                                        <button onClick={() => { updateStatus(option); document.activeElement.blur(); }}>
                                            {Format.statusLabel(option, manga.type)}
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
                        <div key={chapter.id} className="flex items-center gap-3 p-3 rounded-lg bg-base-200">
                            <button
                                className={`btn btn-circle btn-xs ${readChapters.has(chapter.id) ? "btn-primary" : "btn-outline"}`}
                                onClick={() => toggleChapter(chapter)}
                            >
                                <Check size={10} strokeWidth={3} />
                            </button>
                            <NavLink to={`/manga/${id}/chapters/${chapter.id}/read`} className="text-sm w-24 shrink-0">
                                {chapter.title ?? `Chapter ${chapter.chapter_number}`}
                            </NavLink>
                            {chapter.page_count && (
                                <div className="relative flex-1 max-w-48">
                                    <progress
                                        className="progress progress-primary w-full"
                                        value={pageProgress[chapter.id] ?? chapter.last_page_read ?? 0}
                                        max={chapter.page_count}
                                    />
                                    <span className="absolute inset-0 flex items-center justify-center text-xs font-bold">
                                        {(pageProgress[chapter.id] ?? 0) + 1} / {chapter.page_count}
                                    </span>
                                </div>
                            )}
                        </div>
                                            ))}
                </div>
            </>}
        </div>
    )
}