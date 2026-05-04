import Format from "../../utils/format";
import Rating from "../Rating";
import { mediaStatusBadge } from "../../utils/status";

export default function MovieCard({item}) {
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