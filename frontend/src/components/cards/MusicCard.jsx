
/**
 * Display music information in a card
 * @param {any} item the music this displays
 * @returns
 */
export default function MusicCard({item}) {
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