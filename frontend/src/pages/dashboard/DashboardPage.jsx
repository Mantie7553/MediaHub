import { useEffect, useState } from "react"
import api from "../../services/api"
import ContentList from "../../components/layout/ContentList";

/**
 * Dashboard page layout
 * @returns
 */
export default function DashboardPage() {
    const [userContent, setUserContent] = useState([]);
    const [libraryContent, setLibraryContent] = useState([]);
    const [error, setError] = useState("");

    useEffect(() => {
        api.get("/me/media")
        .then(resp => setUserContent(resp.data))
        .catch(err => setError(err.message ?? "Unable to retrieve user tracked media"));
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

    const serverAnime = libraryContent.filter(item => item.type === "anime");
    const serverMovies = libraryContent.filter(item => item.type === "movie");
    const serverMusic = libraryContent.filter(item => item.type === "music_track");
    const serverManga = libraryContent.filter(item => item.type === "manga");

    if (error) return <Error error={error} />

    return <div className="pt-4 flex flex-col gap-10">
        <section className="collapse collapse-arrow border border-base-300">
            <input type="checkbox" defaultChecked/>
            <h2 className="collapse-title font-bold text-xl"><span className="border-l-4 border-primary pl-2">My Collection</span></h2>
            <div className="pl-4 collapse-content">
                <ContentList items={userAnime} heading="Anime" />
                <ContentList items={userMovies} heading="Movies" />
                <ContentList items={userManga} heading="Manga"/>
                <ContentList items={userMusic} heading="Music" />
            </div>
        </section>
        <section className="collapse collapse-arrow border border-base-300">
            <input type="checkbox" defaultChecked/>
            <h2 className="collapse-title font-bold text-xl"><span className="border-l-4 border-primary pl-2">Available Now</span></h2>
            <div className="pl-4 collapse-content">
                <ContentList items={serverAnime} heading="Anime" />
                <ContentList items={serverMovies} heading="Movies" />
                <ContentList items={serverManga} heading="Manga"/>
                <ContentList items={serverMusic} heading="Music" />
            </div>
        </section>
    </div>
}