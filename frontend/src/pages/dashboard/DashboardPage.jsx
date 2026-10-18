import useApi from "../../hooks/useApi";
import ContentList from "../../components/layout/ContentList";
import useUserContent from "../../hooks/useUserContent";
import AlbumList from "../../components/layout/AlbumList";
import ErrorState from "../../components/states/ErrorState";

const sortOrder = (a, b) => (a.title ?? "").localeCompare(b.title ?? "");

/**
 * Dashboard page layout
 * @returns
 */
export default function DashboardPage() {
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

    return <div className="pt-4 flex flex-col gap-10">
        <section className="collapse collapse-arrow border border-base-300">
            <input type="checkbox" defaultChecked/>
            <h2 className="collapse-title font-bold text-xl"><span className="border-l-4 border-primary pl-2">My Collection</span></h2>
            <div className="pl-3 collapse-content">
                {userError ? (
                    <ErrorState compact error={userError} onRetry={retry}/>
                ) : <>
                    <ContentList items={userAnime} heading="Anime" {...tracked}/>
                    <ContentList items={userMovies} heading="Movies" {...tracked}/>
                    <ContentList items={userManga} heading="Manga" {...tracked}/>
                    <AlbumList albums={userMusic} heading="Music" loading={userLoading} emptyText="No albums saved yet" emptyAction={tracked.emptyAction}/>
                    <ContentList items={userLightNovels} heading="Light Novels" {...tracked}/>
                </>}
            </div>
        </section>
        <section className="collapse collapse-arrow border border-base-300">
            <input type="checkbox" defaultChecked/>
            <h2 className="collapse-title font-bold text-xl"><span className="border-l-4 border-primary pl-2">Available Now</span></h2>
            <div className="pl-3 collapse-content">
                {library.error ? (
                    <ErrorState compact error={library.error} onRetry={library.reload}/>
                ) : <>
                    <ContentList items={serverAnime} heading="Anime" {...available}/>
                    <ContentList items={serverMovies} heading="Movies" {...available}/>
                    <ContentList items={serverManga} heading="Manga" {...available}/>
                </>}
                {albums.error ? (
                    <div className="my-4">
                        <h2 className="font-bold mb-2">Music</h2>
                        <ErrorState compact error={albums.error} onRetry={albums.reload}/>
                    </div>
                ) : (
                    <AlbumList albums={albums.data} heading="Music" loading={albums.loading} emptyText="No music yet" emptyHint={available.emptyHint}/>
                )}
                {!library.error && <ContentList items={serverLightNovels} heading="Light Novels" {...available}/>}
            </div>
        </section>
    </div>
}
