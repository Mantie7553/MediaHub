import { useEffect } from "react";
import { Link } from "react-router-dom";
import { Compass, SearchX, Sparkles } from "lucide-react";
import useApi from "../../hooks/useApi";
import useUserContent from "../../hooks/useUserContent";
import useWishlistStore from "../../stores/useWishlistStore";
import { WISHLIST_TYPES } from "../../utils/wishlist";
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
import { albumSorts, artistSorts, collectionSorts, continueSorts, featuredSorts, librarySorts, trackedAlbumSorts, trackedSorts, wishlistSorts } from "../../utils/sortOptions";
import { homePage } from "../../utils/homePages";
import { onCollectionsChanged } from "../../services/collectionsService";

/**
 * One type's home page: what the user is partway through and tracks of that type, then
 * everything of that type on the server. Music adds playlists, collections and artists
 * @param {string} pageKey a homePages key other than "overview"
 */
export default function TypeHomePage({ pageKey }) {
    const page = homePage(pageKey);
    const isMusic = page.key === "music";
    const { userContent, userContentMap, loading: userLoading, error: userError, refresh, retry } = useUserContent();

    // music has no Continue: albums are played, not worked through
    const continuing = useApi(isMusic ? null : `/me/continue?type=${page.type}`, { initial: [] });
    // the titles of this type the user asked to have downloaded, loaded by useUserContent
    const wishlist = useWishlistStore();
    const hasWishlist = WISHLIST_TYPES.includes(page.type);
    const library = useApi(isMusic ? "/albums" : `/media?available=true&type=${page.type}`, { initial: [] });
    // music only: the user's playlists and collections, featured playlists and one playlist per artist
    const collections = useApi(isMusic ? "/me/collections" : null, { initial: [] });
    const featured = useApi(isMusic ? "/featured" : null, { initial: [] });
    const artistPlaylists = useApi(isMusic ? "/artist-playlists" : null, { initial: [] });
    // adding something from a card's picker, or making a new one there, changes the rows
    const reloadCollections = collections.reload;
    useEffect(() => isMusic ? onCollectionsChanged(() => reloadCollections(true)) : undefined, [isMusic, reloadCollections]);

    const { query, setQuery, searchText } = useSearchParamQuery();
    const searching = searchText !== "";
    const matching = items => filterByQuery(items, searchText);

    const mine = matching(trackedOfType(userContent, page.type));
    const available = matching(library.data);
    const wishlisted = hasWishlist ? matching(wishlist.entries.filter(item => item.media_type === page.type)) : [];
    const continueItems = matching(continuing.data);
    const lists = splitCollections(collections.data);
    const playlists = matching(lists.playlists);
    const userCollections = matching(lists.collections);
    const featuredPlaylists = matching(featured.data.filter(c => c.kind === "playlist"));
    const artists = matching(artistPlaylists.data);

    const discoverLink = `/discover?type=${page.discover}`;
    const mineEmpty = searching ? `None of your ${page.noun} match "${searchText}"` : `No ${page.noun} tracked yet`;
    const availableEmpty = searching ? `No ${page.noun} match "${searchText}"` : `No ${page.noun} available yet`;
    const availableHint = searching ? undefined : "Titles show up here once they finish downloading.";

    const mineRows = isMusic ? [
        { key: "albums", label: "Albums", count: mine.length,
            render: () => <AlbumList expandKey="home:music:mine" albums={mine} heading="Albums" sortOptions={trackedAlbumSorts} loading={userLoading} emptyText={mineEmpty}/> },
        { key: "playlist", label: "Playlists", count: playlists.length, error: collections.error, reload: collections.reload,
            render: () => <CollectionList expandKey="home:music:playlists" collections={playlists} kind="playlist" heading="Playlists" sortOptions={collectionSorts}
                loading={collections.loading} emptyText={searching ? mineEmpty : undefined}/> },
        { key: "collection", label: "Collections", count: userCollections.length, error: collections.error, reload: collections.reload,
            render: () => <CollectionList expandKey="home:music:collections" collections={userCollections} kind="collection" heading="Collections" sortOptions={collectionSorts}
                loading={collections.loading} emptyText={searching ? mineEmpty : undefined}/> },
    ] : [
        { key: "mine", label: page.label, count: mine.length,
            render: () => <ContentList expandKey={`home:${page.key}:mine`} items={mine} heading={page.label} sortOptions={trackedSorts(page.type)} loading={userLoading}
                userContentMap={userContentMap} onListChange={refresh} emptyText={mineEmpty}/> },
        ...(hasWishlist ? [{ key: "wishlist", label: "Wishlist", count: wishlisted.length, error: wishlist.error, reload: wishlist.load,
            render: () => <ContentList expandKey={`home:${page.key}:wishlist`} items={wishlisted} heading="Wishlist" sortOptions={wishlistSorts} loading={wishlist.loading}
                userContentMap={userContentMap} onListChange={refresh}
                emptyText={searching ? `Nothing on your wishlist matches "${searchText}"` : `No ${page.noun} wishlisted yet`}/> }] : []),
    ];

    const availableRows = isMusic ? [
        { key: "albums", label: "Albums", count: available.length, error: library.error, reload: library.reload,
            render: () => <AlbumList expandKey="home:music:available" albums={available} heading="Albums" sortOptions={albumSorts} userContentMap={userContentMap} loading={library.loading}
                emptyText={availableEmpty} emptyHint={availableHint}/> },
        { key: "artists", label: "Artists", count: artists.length, error: artistPlaylists.error, reload: artistPlaylists.reload,
            render: () => <CollectionList expandKey="home:music:artists" collections={artists} kind="playlist" heading="Artists" sortOptions={artistSorts}
                loading={artistPlaylists.loading} canCreate={false} emptyText={availableEmpty}/> },
    ] : [
        { key: "available", label: page.label, count: available.length, error: library.error, reload: library.reload,
            render: () => <ContentList expandKey={`home:${page.key}:available`} items={available} heading={page.label} sortOptions={librarySorts(page.type)} loading={library.loading}
                userContentMap={userContentMap} onListChange={refresh} emptyText={availableEmpty} emptyHint={availableHint}/> },
    ];

    const discoverButton = <Link to={discoverLink} className="btn btn-sm btn-ghost rounded-full bg-white/5 gap-1.5">
        <Compass size={14}/> Find more
    </Link>;

    return <div className="flex flex-col gap-10 sm:gap-14">
        <div>
            <PageHeader title={page.label} subtitle={`Your ${page.noun}, and everything available now.`}
                actions={<MakeHomeButton pageKey={page.key}/>}/>
            <HomeTabs current={page.key}/>
        </div>

        {/* a direct child of the page so it can stick for the whole scroll */}
        <HidingSearchBar value={query} onChange={setQuery} placeholder={`Search ${page.noun}…`} className="-mt-4 sm:-mt-8"/>

        {!continuing.loading && continueItems.length > 0 && (
            <ContentList items={continueItems} heading="Continue" expandKey={`home:${page.key}:continue`} sortOptions={continueSorts} userContentMap={userContentMap}
                onListChange={() => { refresh(); continuing.reload(true); }}/>
        )}

        {!featured.loading && featuredPlaylists.length > 0 && (
            <CollectionList collections={featuredPlaylists} heading="Featured" expandKey="home:music:featured" sortOptions={featuredSorts} canCreate={false}/>
        )}

        <HomeSection
            title={`My ${page.label}`}
            loading={userLoading || collections.loading}
            rows={mineRows}
            error={userError}
            onRetry={retry}
            empty={<EmptyState compact icon={searching ? SearchX : Sparkles} title={mineEmpty}
                hint={searching ? undefined : `Add ${page.noun} below, or find more in Discover.`}
                action={searching ? undefined : { label: `Discover ${page.noun}`, to: discoverLink }}/>}
        />

        <HomeSection
            title="Available Now"
            loading={library.loading || artistPlaylists.loading}
            rows={availableRows}
            actions={discoverButton}
            empty={<EmptyState compact icon={searching ? SearchX : Sparkles} title={availableEmpty} hint={availableHint}
                action={searching ? undefined : { label: `Find ${page.noun} in Discover`, to: discoverLink }}/>}
        />
    </div>
}
