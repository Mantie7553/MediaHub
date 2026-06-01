import { Link } from "react-router-dom";
import Format from "../../utils/format";
import Rating from "../Rating";
import { mediaStatusBadge } from "../../utils/status";

export default function Card({item}) {

    let info;
    let path = "";

    switch(item.media_type) {
        case "anime": 
            const season = item.season_number ? `S${item.season_number}` : null
            const total = item.episode_count
            const episodes = item.episodes_watched ?? null
            const progressLabel = episodes !== null 
                ? [season, `E${episodes}${total ? ` / ${total}` : ""}`].filter(Boolean).join(" · ")
                : null
            const progressPct = total ? Math.round((episodes / total) * 100) : 0

            info = <>
                    <span>{progressLabel}</span>
                    <progress className="progress" value={progressPct} max="100"></progress>
                </>
            path = "/anime/"
            break
        case "movie":
            info = <>
                <span>{Format.year(item.release_date)}</span>
            </>
            path = "/watch/"
            break
        case "manga":
            // info = <>
            //     <span>{`Chapter ${item.last_chapter}`}</span>
            // </>
            path = "/manga/"
            break
        case "music": 
            info = <>
                <span>{item.artist}</span>
            </>
            break

        default: break
    }

    const card = (
        <li className="card border border-base-300 w-48 shrink-0">
            <figure className="relative">
                {item.status && 
                    <span className={`badge ${mediaStatusBadge(item.status)} absolute top-2 left-2 z-10 text-xs p-1`}>
                        {Format.cleanString(item.status)}
                    </span>}
                {item.cover_image_url ? (
                    <img src={item.cover_image_url} className="w-full h-48"/>
                ) : (
                    <div className="skeleton h-48 w-full"></div>
                )}
            </figure>
            <div  className="card-body">
                <h3 className="card-title text-sm">{item.media_title ?? item.title}</h3>
                {info}
                <Rating selected={item.rating} id={item.id} />
            </div>
        </li>
    )

    return <Link to={`${path}${item.media_item_id || item.id}`}>{card}</Link>
}