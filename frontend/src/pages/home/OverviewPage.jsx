import { useEffect } from "react";
import { Library, SearchX } from "lucide-react";
import useApi from "../../hooks/useApi";
import useUserContent from "../../hooks/useUserContent";
import useMe from "../../hooks/useMe";
import useSearchParamQuery from "../../hooks/useSearchParamQuery";
import ContentList from "../../components/layout/ContentList";
import AlbumList from "../../components/layout/AlbumList";
import CollectionList from "../../components/layout/CollectionList";
import EmptyState from "../../components/states/EmptyState";
import PageHeader from "../../components/layout/PageHeader";
import HidingSearchBar from "../../components/HidingSearchBar";
import HomeTabs from "../../components/layout/HomeTabs";
import MakeHomeButton from "../../components/layout/MakeHomeButton";
import HomeSection from "./HomeSection";
import { splitCollections, trackedOfType } from "./rows";
import { filterByQuery } from "../../utils/search";
import { onCollectionsChanged } from "../../services/collectionsService";

/**
 * The overview home page: only what the user follows. What they're partway through, what admins
 * featured, and everything on their lists. The server's library lives on each type's home page
 * and in Discover
 */
export default function OverviewPage() {
    const { me } = useMe();
    const { userContent, userContentMap, loading: userLoading, error: userError, refresh, retry } = useUserContent();
    // titles started and not finished, newest activity first
    const continuing = useApi("/me/continue", { initial: [] });
    // the user's playlists (the two that fill themselves first) and collections
    const collections = useApi("/me/collections", { initial: [] });
    // playlists and collections admins picked for everyone
    const featured = useApi("/featured", { initial: [] });
    // adding something from a card's picker, or making a new one there, changes the rows
    const reloadCollections = collections.reload;
    useEffect(() => onCollectionsChanged(() => reloadCollections(true)), [reloadCollections]);

    const { query, setQuery, searchText } = useSearchParamQuery();
    const searching = searchText !== "";
    const matching = items => filterByQuery(items, searchText);

    const anime = matching(trackedOfType(userContent, "anime"));
    const movies = matching(trackedOfType(userContent, "movie"));
    const manga = matching(trackedOfType(userContent, "manga"));
    const music = matching(trackedOfType(userContent, "music_album"));
    const lightNovels = matching(trackedOfType(userContent, "light_novel"));
    const lists = splitCollections(collections.data);
    const playlists = matching(lists.playlists);
    const userCollections = matching(lists.collections);
    const continueItems = matching(continuing.data);
    const featuredItems = matching(featured.data);

    // the same hint on every row, so an empty overview explains itself once per row
    const tracked = {
        loading: userLoading,
        userContentMap,
        onListChange: refresh,
        emptyText: searching ? `Nothing you track matches "${searchText}"` : "Nothing tracked yet",
        emptyAction: searching ? undefined : { label: "Find something in Discover", to: "/discover" },
    };

    const rows = [
        { key: "anime", label: "Anime", count: anime.length, render: () => <ContentList expandKey="home:overview:anime" items={anime} heading="Anime" {...tracked}/> },
        { key: "movie", label: "Movies", count: movies.length, render: () => <ContentList expandKey="home:overview:movie" items={movies} heading="Movies" {...tracked}/> },
        { key: "manga", label: "Manga", count: manga.length, render: () => <ContentList expandKey="home:overview:manga" items={manga} heading="Manga" {...tracked}/> },
        { key: "music", label: "Music", count: music.length, render: () => <AlbumList expandKey="home:overview:music" albums={music} heading="Music" loading={userLoading} emptyText={searching ? tracked.emptyText : "No albums saved yet"} emptyAction={tracked.emptyAction}/> },
        { key: "light_novel", label: "Light Novels", count: lightNovels.length, render: () => <ContentList expandKey="home:overview:light_novel" items={lightNovels} heading="Light Novels" {...tracked}/> },
        // like the other rows these show under "All" once they have something; their filter chips
        // show them empty too, with the card that makes a first one
        { key: "playlist", label: "Playlists", count: playlists.length, error: collections.error, reload: collections.reload,
            render: () => <CollectionList expandKey="home:overview:playlist" collections={playlists} kind="playlist" heading="Playlists" loading={collections.loading}
                emptyText={searching ? tracked.emptyText : undefined}/> },
        { key: "collection", label: "Collections", count: userCollections.length, error: collections.error, reload: collections.reload,
            render: () => <CollectionList expandKey="home:overview:collection" collections={userCollections} kind="collection" heading="Collections" loading={collections.loading}
                emptyText={searching ? tracked.emptyText : undefined}/> },
    ];

    return <div className="flex flex-col gap-10 sm:gap-14">
        <div>
            <PageHeader
                title={me?.username ? <>Welcome back, <span className="text-primary">{me.username}</span></> : "Welcome back"}
                subtitle="Pick up where you left off."
                actions={<MakeHomeButton pageKey="overview"/>}
            />
            <HomeTabs current="overview"/>
        </div>

        {/* pulled up toward the tabs, the page's gap is meant for the rows. It has to stay a
            direct child of the page so it can stick for the whole scroll */}
        <HidingSearchBar value={query} onChange={setQuery} placeholder="Search what you follow…" className="-mt-4 sm:-mt-8"/>

        {!continuing.loading && continueItems.length > 0 && (
            <ContentList items={continueItems} heading="Continue" expandKey="home:overview:continue" userContentMap={userContentMap}
                onListChange={() => { refresh(); continuing.reload(true); }}/>
        )}

        {!featured.loading && featuredItems.length > 0 && (
            <CollectionList collections={featuredItems} heading="Featured" expandKey="home:overview:featured" canCreate={false}/>
        )}

        <HomeSection
            title="My Collection"
            loading={userLoading}
            rows={rows}
            error={userError}
            onRetry={retry}
            empty={searching
                ? <EmptyState compact icon={SearchX} title={tracked.emptyText} hint="Check the spelling, or search Discover to add it."/>
                : <EmptyState compact icon={Library} title="Nothing tracked yet" hint="Add titles from Discover, or browse what's available now from the tabs above." action={tracked.emptyAction}/>}
        />
    </div>
}
