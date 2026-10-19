import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import api from "../../services/api";
import ErrorState from "../../components/states/ErrorState";
import NotFound from "../../components/states/NotFound";
import EmptyState from "../../components/states/EmptyState";
import { DetailsSkeleton, ListSkeleton } from "../../components/states/Skeletons";
import { toast } from "../../stores/useToastStore";
import { openViewer } from "../../utils/navigation";
import { useMediaItem, useUserContent, useRemoveFromList } from "../../hooks";
import DeleteFromServerButton from "../../components/DeleteFromServerButton";
import { animeBadge } from "../../utils/status";
import Format from "../../utils/format";
import { CheckCheck, Play, RotateCcw, Undo2 } from "lucide-react";
import DetailsHero, { HeroChip } from "../../components/details/DetailsHero";
import { ActionMenu, ActionMenuItem, ListStatusMenu } from "../../components/details/Menus";
import ItemRow, { ItemList } from "../../components/details/ItemRow";
import SegmentedTabs from "../../components/SegmentedTabs";

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
    // the season tab picked, null until the user picks one (then the next episode's season shows)
    const [pickedSeason, setPickedSeason] = useState(null);

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
    const seasonNumbers = Object.keys(seasons)
    const multiSeason = seasonNumbers.length > 1

    const positionOf = ep => episodeProgress[ep.id]?.position ?? ep.position_secs ?? 0
    const durationOf = ep => episodeProgress[ep.id]?.duration ?? ep.duration_secs ?? 0
    const watch = (ep, position) => openViewer(navigate, `/watch/episode/${ep.id}`, { position, animeId: id, episodes: episodes })

    // the first episode not watched yet is where the user is up to
    const nextEpisode = episodes.find(ep => !watchedEpisodes.has(ep.id))
    const episodeLabel = ep => multiSeason ? `S${ep.season_number} · E${ep.episode_number}` : `Episode ${ep.episode_number}`
    const primary = nextEpisode
        ? { label: `${positionOf(nextEpisode) > 30 ? "Resume" : "Play"} ${episodeLabel(nextEpisode)}`, run: () => watch(nextEpisode, positionOf(nextEpisode)) }
        : episodes.length > 0
            ? { label: "Watch again", run: () => watch(episodes[0], 0) }
            : null

    const season = pickedSeason ?? String(nextEpisode?.season_number ?? seasonNumbers[0])
    const seasonEpisodes = seasons[season] ?? []

    return (
        <div className="flex flex-col gap-8">
            <DetailsHero
                cover={{ src: anime.cover_image_url }}
                eyebrow="Anime"
                title={anime.title}
                badges={anime.metadata.status && <HeroChip label={anime.metadata.status} tone={animeBadge(anime.metadata.status)?.replace("badge-", "")}/>}
                meta={[
                    anime.release_date && Format.year(anime.release_date),
                    episodes.length > 0 && `${episodes.length} episodes`,
                    multiSeason && `${seasonNumbers.length} seasons`,
                ]}
                genres={anime.metadata.genres ?? []}
                description={anime.description}
                html
                actions={<>
                    {primary && <button className="btn btn-primary btn-lg shadow-glow px-6" onClick={primary.run}>
                        <Play size={18} fill="currentColor"/> {primary.label}
                    </button>}
                    <ListStatusMenu
                        status={userEntry?.status}
                        type={anime.type}
                        options={["current", "completed", "dropped", "planned"]}
                        onSelect={updateStatus}
                        onRemove={removeEntry}
                    />
                    <ActionMenu>
                        <ActionMenuItem icon={CheckCheck} label="Mark all watched" onClick={() => markShow(true)}/>
                        <ActionMenuItem icon={Undo2} label="Mark all unwatched" onClick={() => markShow(false)}/>
                        <DeleteFromServerButton asMenuItem item={{ id, title: anime.title, type: anime.type }} />
                    </ActionMenu>
                </>}
            />

            {episodesError ? (
                <ErrorState compact error={episodesError} onRetry={retryEpisodes} />
            ) : !episodesLoaded ? (
                <ListSkeleton />
            ) : episodes.length === 0 ? (
                <EmptyState compact title="No episodes yet" hint="Episodes show up here once they've downloaded." />
            ) : (
                <ItemList
                    title={multiSeason
                        ? <SegmentedTabs
                            label="Season"
                            value={season}
                            onChange={setPickedSeason}
                            options={seasonNumbers.map(n => ({ key: n, label: `Season ${n}` }))}
                        />
                        : "Episodes"}
                    actions={multiSeason && <ActionMenu label={`Season ${season} actions`} size="sm">
                        <ActionMenuItem icon={CheckCheck} label={`Mark season ${season} watched`} onClick={() => markSeason(season, seasonEpisodes, true)}/>
                        <ActionMenuItem icon={RotateCcw} label={`Mark season ${season} unwatched`} onClick={() => markSeason(season, seasonEpisodes, false)}/>
                    </ActionMenu>}
                >
                    {seasonEpisodes.map(ep => {
                        const watched = watchedEpisodes.has(ep.id)
                        const position = positionOf(ep)
                        const duration = durationOf(ep)
                        const started = duration > 0 && position > 0 && !watched
                        return <ItemRow
                            key={ep.id}
                            checked={watched}
                            onToggle={() => toggleEpisode(ep)}
                            toggleLabel={`Mark episode ${ep.episode_number} ${watched ? "unwatched" : "watched"}`}
                            number={String(ep.episode_number).padStart(2, "0")}
                            title={ep.title ?? `Episode ${ep.episode_number}`}
                            progress={started ? Math.min(position / duration, 1) : null}
                            progressLabel={started
                                ? `${Math.max(1, Math.ceil((duration - position) / 60))} min left`
                                : watched ? "Watched" : duration > 0 ? `${Math.round(duration / 60)} min` : null}
                            current={nextEpisode?.id === ep.id}
                            onOpen={() => watch(ep, position)}
                            openLabel={`Watch episode ${ep.episode_number}`}
                        />
                    })}
                </ItemList>
            )}
        </div>
    )
}
