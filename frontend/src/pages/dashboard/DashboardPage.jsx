import { useEffect, useState } from "react"
import api from "../../services/api"
import Format from "../../utils/format";
import { mediaStatusBadge } from "../../utils/status";

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

    return <section>
        {anime.length > 0 && <ContentList items={anime} heading="Anime" />}
        {movies.length > 0 && <ContentList items={movies} heading="Movies" />}
        {music.length > 0 && <ContentList items={music} heading="Music" />}
    </section>
}


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
                    : <MusicCard key={item.id} item={item} />
            })}
        </ul>
    </div>
}

function TVCard({item}) {
    const season = item.season_number ? `S${item.season_number}` : null
    const episodes = item.episodes_watched ?? 0
    const total = item.episode_count
    const progressPct = total ? Math.round((episodes / total) * 100) : 0
    const progressLabel = [season, `E${episodes}${total ? ` / ${total}` : ""}`].filter(Boolean).join(" · ")

    return <li className="card border border-base-300 w-48 shrink-0">
        <figure className="relative">
            <span className={`badge ${mediaStatusBadge(item.status)} absolute top-2 left-2 z-10 text-xs p-1`}>{Format.cleanString(item.status)}</span>
            {item.cover_image_url ? (
                <img src={item.cover_image_url} className="w-full h-48"/>
            ) : (
                <div className="skeleton h-48 w-full"></div>
            )}
        </figure>
        <div className="card-body">
            <h3 className="card-title">{item.media_title}</h3>
            <span>{progressLabel}</span>
            <progress className="progress" value={progressPct} max="100"></progress>
            <Rating selected={item.rating} id={item.id} />
        </div>
    </li>
}

function MovieCard({item}) {
    return <li className="card border border-base-300 w-48 shrink-0">
        <figure className="relative">
            <span className={`badge ${mediaStatusBadge(item.status)} absolute top-2 left-2 z-10 text-xs p-1`}>{Format.cleanString(item.status)}</span>
            {item.cover_image_url ? (
                <img src={item.cover_image_url} className="w-full h-48"/>
            ) : (
                <div className="skeleton h-48 w-full"></div>
            )}
        </figure>
        <div className="card-body">
            <h3 className="card-title">{item.media_title}</h3>
            <span>{Format.year(item.release_date)}</span>
            <Rating selected={item.rating} id={item.id} />
        </div>
    </li>
}

function MusicCard({item}) {
    return <li className="card border border-base-300 w-48 shrink-0">
        <figure>
            {item.cover_image_url ? (
                <img src={item.cover_image_url}  className="w-full h-32"/>
            ) : (
                <div className="skeleton h-32 w-32"></div>
            )}
        </figure>
        <div className="card-body">
            <h3 className="card-title">{item.media_title}</h3>
            <span>{item.artist}</span>
        </div>
    </li>
}


function Rating({selected, id}) {
    return <div className="rating rating-xs">
        <div className="mask mask-star-2" aria-label="1 star" aria-current={selected === 1}></div>
        <div className="mask mask-star-2" aria-label="2 star" aria-current={selected === 2}></div>
        <div className="mask mask-star-2" aria-label="3 star" aria-current={selected === 3}></div>
        <div className="mask mask-star-2" aria-label="4 star" aria-current={selected === 4}></div>
        <div className="mask mask-star-2" aria-label="5 star" aria-current={selected === 5}></div>
    </div>
}