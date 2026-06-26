import { useEffect, useState } from "react"
import api from "../../services/api"
import ContentList from "../../components/layout/ContentList";
import useUserContent from "../../hooks/useUserContent";
import AlbumList from "../../components/layout/AlbumList";

/**
 * Dashboard page layout
 * @returns
 */
export default function DashboardPage() {
    const { userContent, userContentMap, refresh } = useUserContent();
    const [libraryContent, setLibraryContent] = useState([]);
    const [error, setError] = useState("");
    const [albums, setAlbums] = useState([])

    useEffect(() => {
        api.get("/albums")
            .then(res => setAlbums(res.data ?? []))
            .catch(() => {})
    }, [])

    useEffect(() => {
        api.get("/media?available=true")
        .then(resp => setLibraryContent(resp.data))
        .catch(err => setError(err.message ?? "Unable to retrieve server library"));
    }, [])

    const userAnime = userContent.filter(item => item.media_type === "anime");
    const userMovies = userContent.filter(item => item.media_type === "movie");
    const userMusic = userContent.filter(item => item.media_type === "music_track");
    const userManga = userContent.filter(item => item.media_type === "manga");
    const userLightNovels = userContent.filter(item => item.media_type === "light_novel");

    const serverAnime = libraryContent.filter(item => item.type === "anime");
    const serverMovies = libraryContent.filter(item => item.type === "movie");
    const serverMusic = libraryContent.filter(item => item.type === "music_track");
    const serverManga = libraryContent.filter(item => item.type === "manga");
    const serverLightNovels = libraryContent.filter(item => item.type === "light_novel");

    if (error) return <Error error={error} />

    return <div className="pt-4 flex flex-col gap-10">
        <section className="collapse collapse-arrow border border-base-300">
            <input type="checkbox" defaultChecked/>
            <h2 className="collapse-title font-bold text-xl"><span className="border-l-4 border-primary pl-2">My Collection</span></h2>
            <div className="pl-3 collapse-content">
                <ContentList items={userAnime} heading="Anime" userContentMap={userContentMap} onListChange={refresh}/>
                <ContentList items={userMovies} heading="Movies" userContentMap={userContentMap} onListChange={refresh}/>
                <ContentList items={userManga} heading="Manga" userContentMap={userContentMap} onListChange={refresh}/>
                <AlbumList albums={[]} heading="Music" />
                <ContentList items={userLightNovels} heading="Light Novels" userContentMap={userContentMap} onListChange={refresh}/>

            </div>
        </section>
        <section className="collapse collapse-arrow border border-base-300">
            <input type="checkbox" defaultChecked/>
            <h2 className="collapse-title font-bold text-xl"><span className="border-l-4 border-primary pl-2">Available Now</span></h2>
            <div className="pl-3 collapse-content">
                <ContentList items={serverAnime} heading="Anime" userContentMap={userContentMap} onListChange={refresh}/>
                <ContentList items={serverMovies} heading="Movies" userContentMap={userContentMap} onListChange={refresh}/>
                <ContentList items={serverManga} heading="Manga" userContentMap={userContentMap} onListChange={refresh}/>
                <AlbumList albums={albums} heading="Music" />
                <ContentList items={serverLightNovels} heading="Light Novels" userContentMap={userContentMap} onListChange={refresh}/>

            </div>
        </section>
    </div>
}