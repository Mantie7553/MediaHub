import { useState } from "react";
import { Library, Sparkles } from "lucide-react";
import useApi from "../../hooks/useApi";
import ContentList from "../../components/layout/ContentList";
import useUserContent from "../../hooks/useUserContent";
import useMe from "../../hooks/useMe";
import AlbumList from "../../components/layout/AlbumList";
import ErrorState from "../../components/states/ErrorState";
import EmptyState from "../../components/states/EmptyState";
import PageHeader from "../../components/layout/PageHeader";

const sortOrder = (a, b) => (a.title ?? "").localeCompare(b.title ?? "");

const filters = [
    { key: "all", label: "All" },
    { key: "anime", label: "Anime" },
    { key: "movie", label: "Movies" },
    { key: "manga", label: "Manga" },
    { key: "music", label: "Music" },
    { key: "light_novel", label: "Light Novels" },
];

/**
 * Dashboard page layout
 * @returns
 */
export default function DashboardPage() {
    const { me } = useMe();
    const { userContent, userContentMap, loading: userLoading, error: userError, refresh, retry } = useUserContent();
    const library = useApi("/media?available=true", { initial: [] });
    const albums = useApi("/albums", { initial: [] });

    const userAnime = userContent.filter(item => item.media_type === "anime").sort(sortOrder);
    const userMovies = userContent.filter(item => item.media_type === "movie").sort(sortOrder);
    const userMusic = userContent
    .filter(item => item.media_type === "music_album")
        .map(item => ({
            id: item.album_id,
            title: item.media_title,
            artist: item.artist,
            cover_image_url: item.cover_image_url,
            track_count: null,
        }))
        .sort(sortOrder)
    const userManga = userContent.filter(item => item.media_type === "manga").sort(sortOrder);
    const userLightNovels = userContent.filter(item => item.media_type === "light_novel").sort(sortOrder);

    // shows started but not finished this season
    const continueWatching = userAnime.filter(item =>
        item.season_watched > 0 && (!item.season_total || item.season_watched < item.season_total));

    const serverAnime = library.data.filter(item => item.type === "anime").sort(sortOrder);
    const serverMovies = library.data.filter(item => item.type === "movie").sort(sortOrder);
    const serverManga = library.data.filter(item => item.type === "manga").sort(sortOrder);
    const serverLightNovels = library.data.filter(item => item.type === "light_novel").sort(sortOrder);

    // the same hint on every row of a section, so an empty dashboard explains itself once per row
    const tracked = {
        loading: userLoading,
        userContentMap,
        onListChange: refresh,
        emptyText: "Nothing tracked yet",
        emptyAction: { label: "Find something in Discover", to: "/discover" },
    };
    const available = {
        loading: library.loading,
        userContentMap,
        onListChange: refresh,
        emptyText: "Nothing downloaded yet",
        emptyHint: "Titles show up here once they finish downloading.",
    };

    const collectionRows = [
        { key: "anime", count: userAnime.length, render: () => <ContentList items={userAnime} heading="Anime" {...tracked}/> },
        { key: "movie", count: userMovies.length, render: () => <ContentList items={userMovies} heading="Movies" {...tracked}/> },
        { key: "manga", count: userManga.length, render: () => <ContentList items={userManga} heading="Manga" {...tracked}/> },
        { key: "music", count: userMusic.length, render: () => <AlbumList albums={userMusic} heading="Music" loading={userLoading} emptyText="No albums saved yet" emptyAction={tracked.emptyAction}/> },
        { key: "light_novel", count: userLightNovels.length, render: () => <ContentList items={userLightNovels} heading="Light Novels" {...tracked}/> },
    ];

    const availableRows = [
        { key: "anime", count: serverAnime.length, error: library.error, reload: library.reload, render: () => <ContentList items={serverAnime} heading="Anime" {...available}/> },
        { key: "movie", count: serverMovies.length, error: library.error, reload: library.reload, render: () => <ContentList items={serverMovies} heading="Movies" {...available}/> },
        { key: "manga", count: serverManga.length, error: library.error, reload: library.reload, render: () => <ContentList items={serverManga} heading="Manga" {...available}/> },
        { key: "music", count: albums.data.length, error: albums.error, reload: albums.reload, render: () => <AlbumList albums={albums.data} heading="Music" loading={albums.loading} emptyText="No music yet" emptyHint={available.emptyHint}/> },
        { key: "light_novel", count: serverLightNovels.length, error: library.error, reload: library.reload, render: () => <ContentList items={serverLightNovels} heading="Light Novels" {...available}/> },
    ];

    return <div className="flex flex-col gap-14">
        <PageHeader
            title={me?.username ? <>Welcome back, <span className="text-primary">{me.username}</span></> : "Welcome back"}
            subtitle="Pick up where you left off, or find something new."
        />

        {!userLoading && continueWatching.length > 0 && (
            <ContentList items={continueWatching} heading="Continue Watching" userContentMap={userContentMap} onListChange={refresh}/>
        )}

        <DashboardSection
            title="My Collection"
            loading={userLoading}
            rows={collectionRows}
            error={userError}
            onRetry={retry}
            empty={<EmptyState compact icon={Library} title="Nothing tracked yet" hint="Add titles from Discover and they'll show up here." action={tracked.emptyAction}/>}
        />
        <DashboardSection
            title="Available Now"
            loading={library.loading || albums.loading}
            rows={availableRows}
            empty={<EmptyState compact icon={Sparkles} title="Nothing downloaded yet" hint={available.emptyHint}/>}
        />
    </div>
}

/**
 * A titled group of rows with filter chips. Under "All" empty rows are skipped, and an empty
 * section shows one message instead of one per row
 * @param {object[]} rows { key, count, render, error?, reload? }, error and reload for rows that load separately
 * @param {any} error the whole section failed to load
 * @param {any} empty shown under "All" when every row is empty
 */
function DashboardSection({ title, rows, loading, error, onRetry, empty }) {
    const [filter, setFilter] = useState("all");

    const visible = filter === "all"
        ? rows.filter(row => loading || row.error || row.count > 0)
        : rows.filter(row => row.key === filter);

    // rows sharing one failed request show its error once
    const shownErrors = new Set();

    return <section className="flex flex-col gap-6">
        <div className="flex flex-wrap items-center justify-between gap-4 border-b border-white/5 pb-4">
            <h2 className="text-2xl font-semibold tracking-tight flex items-center gap-3">
                <span className="w-1 h-6 rounded-full bg-primary shadow-[0_0_10px_rgb(242_89_18/0.7)]"></span>
                {title}
            </h2>
            <div className="flex flex-wrap gap-1.5" role="tablist" aria-label={`Filter ${title}`}>
                {filters.map(f => {
                    const count = f.key === "all" ? null : rows.find(row => row.key === f.key)?.count;
                    const active = filter === f.key;
                    return <button
                        key={f.key}
                        role="tab"
                        aria-selected={active}
                        onClick={() => setFilter(f.key)}
                        className={`h-8 px-3.5 rounded-full text-sm transition-all duration-150 flex items-center gap-1.5
                            ${active
                                ? "bg-primary text-primary-content shadow-glow font-medium"
                                : "bg-white/5 text-base-content/70 hover:bg-white/10 hover:text-base-content"}`}
                    >
                        {f.label}
                        {!loading && count > 0 && <span className={`text-xs tabular-nums ${active ? "text-primary-content/80" : "text-base-content/40"}`}>{count}</span>}
                    </button>
                })}
            </div>
        </div>

        {error ? (
            <ErrorState compact error={error} onRetry={onRetry}/>
        ) : visible.length === 0 ? (
            empty
        ) : visible.map(row => {
            if (row.error) {
                if (shownErrors.has(row.error)) return null;
                shownErrors.add(row.error);
                return <ErrorState key={row.key} compact error={row.error} onRetry={row.reload}/>;
            }
            return <div key={row.key}>{row.render()}</div>;
        })}
    </section>
}
