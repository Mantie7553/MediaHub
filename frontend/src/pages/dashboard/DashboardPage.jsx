import { useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { Library, SearchX, Sparkles } from "lucide-react";
import useApi from "../../hooks/useApi";
import ContentList from "../../components/layout/ContentList";
import useUserContent from "../../hooks/useUserContent";
import useMe from "../../hooks/useMe";
import AlbumList from "../../components/layout/AlbumList";
import CollectionList from "../../components/layout/CollectionList";
import ErrorState from "../../components/states/ErrorState";
import EmptyState from "../../components/states/EmptyState";
import PageHeader from "../../components/layout/PageHeader";
import SegmentedTabs from "../../components/SegmentedTabs";
import HidingSearchBar from "../../components/HidingSearchBar";
import { filterByQuery } from "../../utils/search";
import { onCollectionsChanged } from "../../services/collectionsService";

// list entries name their title media_title, library items title
const sortOrder = (a, b) => (a.media_title ?? a.title ?? "").localeCompare(b.media_title ?? b.title ?? "");

// how long typing pauses before the search goes into the address
const SEARCH_DELAY_MS = 250;

const filters = [
    { key: "all", label: "All" },
    { key: "anime", label: "Anime" },
    { key: "movie", label: "Movies" },
    { key: "manga", label: "Manga" },
    { key: "music", label: "Music" },
    { key: "light_novel", label: "Light Novels" },
    { key: "playlist", label: "Playlists" },
    { key: "collection", label: "Collections" },
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
    // titles started and not finished, newest activity first
    const continuing = useApi("/me/continue", { initial: [] });
    // the user's playlists (the two that fill themselves first) and collections
    const collections = useApi("/me/collections", { initial: [] });
    // playlists and collections admins picked for everyone
    const featured = useApi("/featured", { initial: [] });
    // adding something from a card's picker, or making a new one there, changes the rows
    const reloadCollections = collections.reload;
    useEffect(() => onCollectionsChanged(() => reloadCollections(true)), [reloadCollections]);

    // the search filters every row as it's typed, and is kept in the address (?q=) once typing
    // pauses, so coming back to the dashboard or refreshing keeps it
    const [params, setParams] = useSearchParams();
    const [query, setQuery] = useState(params.get("q") ?? "");
    const searchText = query.trim();
    const searching = searchText !== "";
    useEffect(() => {
        if (searchText === (params.get("q") ?? "")) return;
        const timer = setTimeout(() => {
            // from the address as it is now: React Router hands updaters the last render's params
            const next = new URLSearchParams(window.location.search);
            if (searchText) next.set("q", searchText);
            else next.delete("q");
            setParams(next, { replace: true });
        }, SEARCH_DELAY_MS);
        return () => clearTimeout(timer);
    }, [searchText, params, setParams]);
    const matching = items => filterByQuery(items, searchText);

    const userAnime = matching(userContent.filter(item => item.media_type === "anime")).sort(sortOrder);
    const userMovies = matching(userContent.filter(item => item.media_type === "movie")).sort(sortOrder);
    const userMusic = matching(userContent.filter(item => item.media_type === "music_album"))
        .map(item => ({
            id: item.album_id,
            title: item.media_title,
            artist: item.artist,
            cover_image_url: item.cover_image_url,
            track_count: null,
        }))
        .sort(sortOrder)
    const userManga = matching(userContent.filter(item => item.media_type === "manga")).sort(sortOrder);
    const userLightNovels = matching(userContent.filter(item => item.media_type === "light_novel")).sort(sortOrder);

    const serverAnime = matching(library.data.filter(item => item.type === "anime")).sort(sortOrder);
    const serverMovies = matching(library.data.filter(item => item.type === "movie")).sort(sortOrder);
    const serverManga = matching(library.data.filter(item => item.type === "manga")).sort(sortOrder);
    const serverLightNovels = matching(library.data.filter(item => item.type === "light_novel")).sort(sortOrder);

    const serverAlbums = matching(albums.data);
    // the playlists that fill themselves are always there, so they show only once something is
    // in them
    const playlists = matching(collections.data.filter(c => c.kind === "playlist" && (!c.smart || c.item_count > 0)));
    const userCollections = matching(collections.data.filter(c => c.kind === "collection"));
    const playlistCount = playlists.length;
    const continueItems = matching(continuing.data);
    const featuredItems = matching(featured.data);

    // the same hint on every row of a section, so an empty dashboard explains itself once per row
    const tracked = {
        loading: userLoading,
        userContentMap,
        onListChange: refresh,
        emptyText: searching ? `Nothing you track matches "${searchText}"` : "Nothing tracked yet",
        emptyAction: searching ? undefined : { label: "Find something in Discover", to: "/discover" },
    };
    const available = {
        loading: library.loading,
        userContentMap,
        onListChange: refresh,
        emptyText: searching ? `Nothing on the server matches "${searchText}"` : "Nothing downloaded yet",
        emptyHint: searching ? undefined : "Titles show up here once they finish downloading.",
    };

    const collectionRows = [
        { key: "anime", count: userAnime.length, render: () => <ContentList items={userAnime} heading="Anime" {...tracked}/> },
        { key: "movie", count: userMovies.length, render: () => <ContentList items={userMovies} heading="Movies" {...tracked}/> },
        { key: "manga", count: userManga.length, render: () => <ContentList items={userManga} heading="Manga" {...tracked}/> },
        { key: "music", count: userMusic.length, render: () => <AlbumList albums={userMusic} heading="Music" loading={userLoading} emptyText={searching ? tracked.emptyText : "No albums saved yet"} emptyAction={tracked.emptyAction}/> },
        { key: "light_novel", count: userLightNovels.length, render: () => <ContentList items={userLightNovels} heading="Light Novels" {...tracked}/> },
        // like the other rows these show under "All" once they have something; their filter chips
        // show them empty too, with the card that makes a first one
        { key: "playlist", count: playlistCount, error: collections.error, reload: collections.reload,
            render: () => <CollectionList collections={playlists} kind="playlist" heading="Playlists" loading={collections.loading}
                emptyText={searching ? tracked.emptyText : undefined}/> },
        { key: "collection", count: userCollections.length, error: collections.error, reload: collections.reload,
            render: () => <CollectionList collections={userCollections} kind="collection" heading="Collections" loading={collections.loading}
                emptyText={searching ? tracked.emptyText : undefined}/> },
    ];

    const availableRows = [
        { key: "anime", count: serverAnime.length, error: library.error, reload: library.reload, render: () => <ContentList items={serverAnime} heading="Anime" {...available}/> },
        { key: "movie", count: serverMovies.length, error: library.error, reload: library.reload, render: () => <ContentList items={serverMovies} heading="Movies" {...available}/> },
        { key: "manga", count: serverManga.length, error: library.error, reload: library.reload, render: () => <ContentList items={serverManga} heading="Manga" {...available}/> },
        { key: "music", count: serverAlbums.length, error: albums.error, reload: albums.reload, render: () => <AlbumList albums={serverAlbums} heading="Music" loading={albums.loading} emptyText={searching ? available.emptyText : "No music yet"} emptyHint={available.emptyHint}/> },
        { key: "light_novel", count: serverLightNovels.length, error: library.error, reload: library.reload, render: () => <ContentList items={serverLightNovels} heading="Light Novels" {...available}/> },
    ];

    return <div className="flex flex-col gap-10 sm:gap-14">
        <PageHeader
            title={me?.username ? <>Welcome back, <span className="text-primary">{me.username}</span></> : "Welcome back"}
            subtitle="Pick up where you left off, or find something new."
        />

        {/* pulled up against the header, the page's gap is meant for the rows. It has to stay a
            direct child of the page so it can stick for the whole scroll */}
        <HidingSearchBar value={query} onChange={setQuery} placeholder="Search your dashboard…" className="-mt-7 sm:-mt-11"/>

        {!continuing.loading && continueItems.length > 0 && (
            <ContentList items={continueItems} heading="Continue" userContentMap={userContentMap}
                onListChange={() => { refresh(); continuing.reload(true); }}/>
        )}

        {!featured.loading && featuredItems.length > 0 && (
            <CollectionList collections={featuredItems} heading="Featured" canCreate={false}/>
        )}

        <DashboardSection
            title="My Collection"
            loading={userLoading}
            rows={collectionRows}
            error={userError}
            onRetry={retry}
            empty={searching
                ? <EmptyState compact icon={SearchX} title={tracked.emptyText} hint="Check the spelling, or search Discover to add it."/>
                : <EmptyState compact icon={Library} title="Nothing tracked yet" hint="Add titles from Discover and they'll show up here." action={tracked.emptyAction}/>}
        />
        <DashboardSection
            title="Available Now"
            loading={library.loading || albums.loading}
            rows={availableRows}
            empty={searching
                ? <EmptyState compact icon={SearchX} title={available.emptyText} hint="Check the spelling, or request it from Discover."/>
                : <EmptyState compact icon={Sparkles} title="Nothing downloaded yet" hint={available.emptyHint}/>}
        />
    </div>
}

/**
 * A titled group of rows with filter chips. Under "All" empty rows are skipped, and an empty
 * section shows one message instead of one per row
 * @param {object[]} rows { key, count, render, error?, reload? }, error and reload for rows that
 *  load separately. Only the filter chips for these rows are offered
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
        <div className="flex flex-col sm:flex-row sm:flex-wrap sm:items-center justify-between gap-3 sm:gap-4 border-b border-white/5 pb-4">
            <h2 className="text-xl sm:text-2xl font-semibold tracking-tight flex items-center gap-3">
                <span className="w-1 h-6 rounded-full bg-primary shadow-[0_0_10px_rgb(242_89_18/0.7)]"></span>
                {title}
            </h2>
            <SegmentedTabs
                options={filters.filter(f => f.key === "all" || rows.some(row => row.key === f.key)).map(f => ({ ...f, count: loading || f.key === "all" ? 0 : rows.find(row => row.key === f.key)?.count }))}
                value={filter}
                onChange={setFilter}
                label={`Filter ${title}`}
            />
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
