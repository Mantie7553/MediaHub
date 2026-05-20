import Format from "../../utils/format";
import Rating from "../Rating";
import { mediaStatusBadge } from "../../utils/status";
import { Link } from "react-router-dom";

/**
 * Display Anime information as a card
 * @param {any} item the anime this displays
 * @returns
 */
export default function TVCard({item}) {
    const season = item.season_number ? `S${item.season_number}` : null
    const episodes = item.episodes_watched ?? 0
    const total = item.episode_count
    const progressPct = total ? Math.round((episodes / total) * 100) : 0
    const progressLabel = [season, `E${episodes}${total ? ` / ${total}` : ""}`].filter(Boolean).join(" · ")

    const card = (
        <li className="card border border-base-300 w-48 shrink-0">
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
    )

    if (!item.plex_rating_key) return card

    return <Link to={`/watch/${item.media_item_id}`}>{card}</Link>
}