import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import api from "../../services/api";
import ErrorState from "../../components/states/ErrorState";
import NotFound from "../../components/states/NotFound";
import EmptyState from "../../components/states/EmptyState";
import { DetailsSkeleton, ListSkeleton } from "../../components/states/Skeletons";
import Cover from "../../components/Cover";
import { toast } from "../../stores/useToastStore";
import { goBack, openViewer } from "../../utils/navigation";
import { useMediaItem, useUserContent, useRemoveFromList } from "../../hooks";
import DeleteFromServerButton from "../../components/DeleteFromServerButton";
import { animeBadge, mediaStatusBadge } from "../../utils/status";
import Format from "../../utils/format";
import { Check, ArrowLeft } from "lucide-react";

export default function AnimeDetailsPage() {
    const { id } = useParams();
    const navigate = useNavigate();
    const { userContentMap, refresh } = useUserContent();
    const userEntry = userContentMap[id];
    const removeEntry = useRemoveFromList(userEntry, refresh);
    const { item: anime, loading, error, notFound, refetch } = useMediaItem(id);
    const [episodes, setEpisodes] = useState([]);
    const [watchedEpisodes, setWatchedEpisodes] = useState(new Set());
    const [episodeProgress, setEpisodeProgress] = useState({});
    const [episodesLoaded, setEpisodesLoaded] = useState(false);
    const [episodesError, setEpisodesError] = useState(null);
    const [episodesAttempt, setEpisodesAttempt] = useState(0);

    useEffect(() => {
        // background: the refresh when the tab regains focus, which fails quietly
        function fetchEpisodes(background) {
            api.get(`/media/${id}/episodes`)
                .then(res => {
                    const data = res.data ?? [];
                    setEpisodes(data);
                    setWatchedEpisodes(new Set(data.filter(ep => ep.watched).map(ep => ep.id)));
                    const initial = {};
                    data.forEach(ep => { initial[ep.id] = { position: ep.position_secs, duration: ep.duration_secs }; });
                    setEpisodeProgress(initial);
                    setEpisodesLoaded(true);
                    setEpisodesError(null);
                })
                .catch(err => { if (!background) setEpisodesError(err) })
        }

        fetchEpisodes(false);
        const onFocus = () => fetchEpisodes(true);
        window.addEventListener("focus", onFocus);
        return () => window.removeEventListener("focus", onFocus);
    }, [id, episodesAttempt])

    function retryEpisodes() {
        setEpisodesError(null);
        setEpisodesAttempt(a => a + 1);
    }

    function updateStatus(status) {
        const request = userEntry
            ? api.put(`/me/media/${userEntry.id}`, { status })
            : api.post(`/me/media`, { media_item_id: id, status });
        request
            .then(() => refresh())
            .catch(err => toast.error(err, "Could not update your list"));
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
            .catch(err => toast.error(err, "Could not update the episode"))
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
            .catch(err => toast.error(err, `Could not update season ${seasonNum}`))
    }

    function markShow(watched) {
        api.put(`/anime/${id}/watched`, { watched })
            .then(() => {
                setWatchedEpisodes(watched ? new Set(episodes.map(ep => ep.id)) : new Set());
                updateStatus(watched ? "completed" : "planned");
            })
            .catch(err => toast.error(err, "Could not update the show"));
    }

    if (loading) return <DetailsSkeleton />
    if (notFound) return <NotFound title="Show not found" hint="This show isn't in the library any more. It may have been removed." />
    if (error) return <ErrorState error={error} title="Couldn't load this show" onRetry={refetch} showBack />
    if (!anime) return null

    const seasons = episodes.reduce((acc, ep) => {
        const s = ep.season_number
        if (!acc[s]) acc[s] = []
        acc[s].push(ep)
        return acc
    }, {})

    return (
        <div className="flex flex-col gap-8">
            <button className="btn btn-ghost btn-sm self-start" onClick={() => goBack(navigate)}>
                <ArrowLeft size={18} strokeWidth={2}/> Back
            </button>
            <div className="flex gap-6">
                <Cover src={anime.cover_image_url} alt={anime.title} className="w-48 h-64 shrink-0 object-cover rounded-md" />
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
                                {userEntry && (
                                    <li className="border-t border-base-300 mt-1 pt-1">
                                        <button className="text-error" onClick={() => { removeEntry(); document.activeElement.blur(); }}>
                                            Remove from list
                                        </button>
                                    </li>
                                )}
                            </ul>
                        </div>
                        <button className="btn btn-outline btn-sm" onClick={() => markShow(true)}>
                            Mark all watched
                        </button>
                        <button className="btn btn-outline btn-sm" onClick={() => markShow(false)}>
                            Mark all unwatched
                        </button>
                        <DeleteFromServerButton item={{ id, title: anime.title, type: anime.type }} />
                    </div>
                </div>
            </div>

            {episodesError ? (
                <ErrorState compact error={episodesError} onRetry={retryEpisodes} />
            ) : !episodesLoaded ? (
                <ListSkeleton />
            ) : episodes.length === 0 ? (
                <EmptyState compact title="No episodes yet" hint="Episodes show up here once they've downloaded." />
            ) : (
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
                                            onClick={() => openViewer(navigate, `/watch/episode/${ep.id}`, { position: episodeProgress[ep.id]?.position ?? ep.position_secs, animeId: id, episodes: episodes }) }
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