import { NavLink } from "react-router-dom";

/**
 * Display manga information in a card
 * @param {any} item the manga this displays
 * @returns
 */
export default function MangaCard({item}) {
    return <li className="card border border-base-300 w-48 shrink-0">
        <NavLink to={`/manga/${item.id}`}>
            <figure>
                {item.cover_image_url ? (
                    <img src={item.cover_image_url}  className="w-full h-48"/>
                ) : (
                    <div className="skeleton h-48 w-full"></div>
                )}
            </figure>
            <div className="card-body">
                <h3 className="card-title">{item.title}</h3>
            </div>
        </NavLink>
    </li>
}