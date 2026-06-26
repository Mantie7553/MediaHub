import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import api from "../../services/api";
import Loading from "../../components/states/Loading";
import Error from "../../components/states/Error";
import { useMediaItem } from "../../hooks";
import { animeBadge } from "../../utils/status";
import Format from "../../utils/format";
import { Check } from "lucide-react";

export default function AnimeDetailsPage() {
    const { id } = useParams();
    const navigate = useNavigate();
    const { item: anime, loading, error } = useMediaItem(id);
    const [episodes, setEpisodes] = useState([]);
    const [watchedEpisodes, setWatchedEpisodes] = useState(new Set());

    useEffect(() => {
        api.get(`/media/${id}/episodes`)
            .then(res => {
                setEpisodes(res.data ?? []);
                setWatchedEpisodes(new Set((res.data ?? []).filter(ep => ep.watched).map(ep => ep.id)));
            })
            .catch(() => {})
    }, [id])

    function toggleEpisode(ep) {
        const watched = !watchedEpisodes.has(ep.id);
        api.put(`/episodes/${ep.id}/watched`, { watched })
            .then(() => {
                setWatchedEpisodes(prev => {
                    const next = new Set(prev);
                    watched ? next.add(ep.id) : next.delete(ep.id);
                    return next;
                });
            })
            .catch(() => {})
    }

    function markSeason(seasonNum, eps, watched) {
        api.put(`/anime/${id}/seasons/${seasonNum}/watched`, { watched })
            .then(() => {
                setWatchedEpisodes(prev => {
                    const next = new Set(prev);
                    eps.forEach(ep => watched ? next.add(ep.id) : next.delete(ep.id));
                    return next;
                });
            })
            .catch(() => {})
    }

    function markShow(watched) {
        api.put(`/anime/${id}/watched`, { watched })
            .then(() => {
                setWatchedEpisodes(watched ? new Set(episodes.map(ep => ep.id)) : new Set());
            })
            .catch(() => {})
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
                                <button className="btn btn-outline btn-xs" onClick={() => markSeason(seasonNum, eps, true)}>
                                    Mark all watched
                                </button>
                                <button className="btn btn-outline btn-xs" onClick={() => markSeason(seasonNum, eps, false)}>
                                    Mark all unwatched
                                </button>
                            </div>
                            <div className="flex flex-col gap-2">
                                {eps.map(ep => (
                                    <div key={ep.id} className="flex items-center justify-between p-3 rounded-lg bg-base-200">
                                        <div className="flex items-center gap-3">
                                            <button
                                                className={`btn btn-circle btn-xs ${watchedEpisodes.has(ep.id) ? "btn-primary" : "btn-outline"}`}
                                                onClick={() => toggleEpisode(ep)}
                                            >
                                                <Check size={10} strokeWidth={3} />
                                            </button>
                                            <span className="text-sm">
                                                EP {ep.episode_number} — {ep.title ?? "Untitled"}
                                            </span>
                                        </div>
                                        <button
                                            className="btn btn-sm btn-primary"
                                            onClick={() => navigate(`/watch/episode/${ep.id}`)}
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