import { useEffect, useState } from "react"
import api from "../../services/api"
import { TVCard, MovieCard, MusicCard, MangaCard } from "../../components/cards"

/**
 * Dashboard page layout
 * @returns
 */
export default function DashboardPage() {
    const [content, setContent] = useState([]);
    const [error, setError] = useState("");

    useEffect(() => {
        api.get("me/media")
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

/**
 * A list of some content type
 * @param {any} items the items this list will contain
 * @param {any} heading the heading for this list
 * @returns
 */
function ContentList({items, heading}) {

    return <div className="my-4 max-w-fit">
        <div className="flex justify-between items-center mb-2">
            <h2 className="font-bold">{heading}</h2>
            <button className="link">Show all</button>
        </div>
        <ul className="flex gap-2 overflow-x-auto flex-nowrap">
            {items.slice(0,10).map(item => {
                return item.media_type === "anime" ? 
                    <TVCard key={item.id} item={item} />
                    : item.media_type === "movie" ? 
                    <MovieCard key={item.id} item={item} />
                    : item.media_type === "manga" ?
                    <MangaCard key={item.id} item={item} />
                    : <MusicCard key={item.id} item={item} />
            })}
        </ul>
    </div>
}