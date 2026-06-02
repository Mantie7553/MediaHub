import { useEffect, useState } from "react"
import api from "../../services/api"
import { Card } from "../../components/cards"
import ContentList from "../../components/layout/ContentList";

/**
 * Dashboard page layout
 * @returns
 */
export default function DashboardPage() {
    const [content, setContent] = useState([]);
    const [error, setError] = useState("");

    useEffect(() => {
        api.get("/me/media")
        .then(resp => setContent(resp.data))
        .catch(err => setError(err.message ?? "Unable to retrieve media"));
    }, [])

    const anime = content.filter(item => item.media_type === "anime");
    const movies = content.filter(item => item.media_type === "movie");
    const music = content.filter(item => item.media_type === "music_track");
    const manga = content.filter(item => item.media_type === "manga")

    if (error) return <Error error={error} />

    return <section>
        <ContentList items={anime} heading="Anime" />
        <ContentList items={movies} heading="Movies" />
        <ContentList items={manga} heading="Manga"/>
        <ContentList items={music} heading="Music" />
    </section>
}