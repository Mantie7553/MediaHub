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

    return <div>
        <section>
            <h2>My Collection</h2>
            <ContentList items={userAnime} heading="Anime" />
            <ContentList items={userMovies} heading="Movies" />
            <ContentList items={userManga} heading="Manga"/>
            <ContentList items={userMusic} heading="Music" />
        </section>
        <section>
            <h2>Available Now</h2>
            <ContentList items={serverAnime} heading="Anime" />
            <ContentList items={serverMovies} heading="Movies" />
            <ContentList items={serverManga} heading="Manga"/>
            <ContentList items={serverMusic} heading="Music" />
        </section>
    </div>
}