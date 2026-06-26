import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import api from "../../services/api";
import Loading from "../../components/states/Loading";
import Error from "../../components/states/Error";
import { useMediaItem, useUserContent } from "../../hooks";
import { animeBadge, mediaStatusBadge } from "../../utils/status";
import Format from "../../utils/format";
import { Check, ArrowLeft } from "lucide-react";

export default function AnimeDetailsPage() {
    const { id } = useParams();
    const navigate = useNavigate();
    const { userContentMap, refresh } = useUserContent();
    const userEntry = userContentMap[id];
    const { item: anime, loading, error } = useMediaItem(id);
    const [episodes, setEpisodes] = useState([]);
    const [watchedEpisodes, setWatchedEpisodes] = useState(new Set());
    const [episodeProgress, setEpisodeProgress] = useState({});

    useEffect(() => {
        function fetchEpisodes() {
            api.get(`/media/${id}/episodes`)
                .then(res => {
                    const data = res.data ?? [];
                    setEpisodes(data);
                    setWatchedEpisodes(new Set(data.filter(ep => ep.watched).map(ep => ep.id)));
                    const initial = {};
                    data.forEach(ep => { initial[ep.id] = { position: ep.position_secs, duration: ep.duration_secs }; });
                    setEpisodeProgress(initial);
                })
                .catch(() => {})
        }

        fetchEpisodes();
        window.addEventListener("focus", fetchEpisodes);
        return () => window.removeEventListener("focus", fetchEpisodes);
    }, [id])

    function updateStatus(status) {
        if (userEntry) {
            api.put(`/me/media/${userEntry.id}`, { status }).then(() => refresh());
        } else {
            api.post(`/me/media`, { media_item_id: id, status }).then(() => refresh());
        }
    }

    function toggleEpisode(ep) {
        const watched = !watchedEpisodes.has(ep.id);
        api.put(`/episodes/${ep.id}/watched`, { watched })
            .then(() => {
                const next = new Set(watchedEpisodes);
                watched ? next.add(ep.id) : next.delete(ep.id);
                const status = next.size === episodes.length ? "completed" : "current";
                setWatchedEpisodes(next);
                updateStatus(status);
            })
            .catch(() => {})
    }

    function markSeason(seasonNum, eps, watched) {
        api.put(`/anime/${id}/seasons/${seasonNum}/watched`, { watched })
            .then(() => {
                const next = new Set(watchedEpisodes);
                eps.forEach(ep => watched ? next.add(ep.id) : next.delete(ep.id));
                const status = next.size === episodes.length ? "completed" : watched ? "current" : "planned";
                setWatchedEpisodes(next);
                updateStatus(status);
            })
            .catch(() => {})
    }

    function markShow(watched) {
        api.put(`/anime/${id}/watched`, { watched })
            .then(() => {
                setWatchedEpisodes(watched ? new Set(episodes.map(ep => ep.id)) : new Set());
                updateStatus(watched ? "completed" : "planned");
            })
            .catch(() => {});
    }

    if (loading) return <Loading />
    if (error) return <Error error={error} />
    if (!anime) return null

    const seasons = episodes.reduce((acc, ep) => {
        const s = ep.season_number
        if (!acc[s]) acc[s] = []
        acc[s].push(ep)
        return acc
    }, {})

    return (
        <div className="flex flex-col gap-8">
            <button className="btn btn-ghost btn-sm self-start" onClick={() => navigate("/")}>
                <ArrowLeft size={18} strokeWidth={2}/> Back
            </button>
            <div className="flex gap-6">
                <img src={anime.cover_image_url} className="w-48 h-64 object-cover rounded-md" />
                <div className="flex flex-col gap-3">
                    <h2 className="text-2xl font-bold">{anime.title}</h2>
                    <span className={`badge ${animeBadge(anime.metadata.status)}`}>{anime.metadata.status}</span>
                    {anime.release_date && (
                        <span className="text-sm text-neutral-content">{Format.date(anime.release_date)}</span>
                    )}
                    <div className="flex flex-wrap gap-1">
                        {(anime.metadata.genres ?? []).map((genre, i) => (
                            <span key={`${anime.title}-${i}`} className="badge">{genre}</span>
                        ))}
                    </div>
                    {anime.description && (
                        <p className="text-sm max-w-xl" dangerouslySetInnerHTML={{ __html: anime.description}} />
                    )}
                    <div className="flex flex-wrap gap-2">
                        <div className="dropdown">
                            <div tabIndex={0} className={`badge ${mediaStatusBadge(userEntry?.status)} cursor-pointer`}>
                                {Format.statusLabel(userEntry?.status, anime.type)}
                            </div>
                            <ul tabIndex={0} className="dropdown-content menu bg-base-200 rounded-box z-10 p-2 shadow gap-1">
                                {["current", "completed", "dropped", "planned"].map(option => (
                                    <li key={option}>
                                        <button onClick={() => { updateStatus(option); document.activeElement.blur(); }}>
                                            {Format.statusLabel(option, anime.type)}
                                        </button>
                                    </li>
                                ))}
                            </ul>
                        </div>
                        <button className="btn btn-outline btn-sm" onClick={() => markShow(true)}>
                            Mark all watched
                        </button>
                        <button className="btn btn-outline btn-sm" onClick={() => markShow(false)}>
                            Mark all unwatched
                        </button>
                    </div>
                </div>
            </div>

            {Object.keys(seasons).length > 0 && (
                <div className="flex flex-col gap-6">
                    {Object.entries(seasons).map(([seasonNum, eps]) => (
                        <div key={seasonNum}>
                            <div className="flex items-center gap-3 mb-3">
                                <h3 className="text-lg font-semibold">Season {seasonNum}</h3>
                                {Object.keys(seasons).length > 1 && <>
                                    <button className="btn btn-outline btn-xs" onClick={() => markSeason(seasonNum, eps, true)}>
                                        Mark all watched
                                    </button>
                                    <button className="btn btn-outline btn-xs" onClick={() => markSeason(seasonNum, eps, false)}>
                                        Mark all unwatched
                                    </button>
                                </>}
                            </div>
                            <div className="flex flex-col gap-2">
                                {eps.map(ep => (
                                    <div key={ep.id} className="flex items-center justify-between p-3 rounded-lg bg-base-200">
                                        <div className="flex items-center gap-3 flex-1">
                                            <button
                                                className={`btn btn-circle btn-xs ${watchedEpisodes.has(ep.id) ? "btn-primary" : "btn-outline"}`}
                                                onClick={() => toggleEpisode(ep)}
                                            >
                                                <Check size={10} strokeWidth={3} />
                                            </button>
                                            <span className="text-sm">
                                                EP {ep.episode_number} — {ep.title ?? "Untitled"}
                                            </span>
                                            {(episodeProgress[ep.id]?.duration ?? ep.duration_secs) > 0 && (
                                                <div className="relative flex-1 max-w-48">
                                                    <progress
                                                        className="progress progress-primary w-full"
                                                        value={episodeProgress[ep.id]?.position ?? ep.position_secs}
                                                        max={episodeProgress[ep.id]?.duration ?? ep.duration_secs}
                                                    />
                                                </div>
                                            )}
                                        </div>
                                        <button
                                            className="btn btn-sm btn-primary"
                                            onClick={() => navigate(`/watch/episode/${ep.id}`, { state: { position: episodeProgress[ep.id]?.position ?? ep.position_secs, animeId: id, episodes: episodes } }) }
                                        >
                                            Watch
                                        </button>
                                    </div>
                                ))}
                            </div>
                        </div>
                    ))}
                </div>
            )}
        </div>
    )
}