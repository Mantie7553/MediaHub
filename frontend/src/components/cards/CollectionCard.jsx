import { Link } from "react-router-dom"
import { Library, ListMusic, Plus, Sparkles } from "lucide-react"
import Cover from "../Cover"

/**
 * The covers of a playlist or collection: a 2x2 mosaic of up to 4, one filling the square, or
 * an icon while it's empty
 * @param {string[]} covers cover URLs from the start of it
 * @param {string} kind "playlist" or "collection", for the empty icon
 * @param {boolean} smart one of the playlists that fill themselves, shown with a sparkle
 */
export function CollectionCover({ covers = [], kind, smart = false, className = "" }) {
    // summaries don't say where a cover is from; Cover only proxies MangaDex's own URLs, so
    // naming the source on all of them is safe
    const source = "mangadex"
    const Icon = smart ? Sparkles : kind === "collection" ? Library : ListMusic
    if (covers.length === 0) {
        return <div className={`grid place-items-center bg-linear-to-br from-primary/25 to-base-300 ${className}`}>
            <Icon size={36} className="text-primary/80"/>
        </div>
    }
    if (covers.length < 4) {
        return <Cover src={covers[0]} source={source} alt="" className={`object-cover ${className}`}/>
    }
    return <div className={`grid grid-cols-2 grid-rows-2 ${className}`}>
        {covers.slice(0, 4).map(src => <Cover key={src} src={src} source={source} alt="" className="size-full object-cover"/>)}
    </div>
}

/**
 * A square card for a playlist or collection, linking to it
 * @param {object} collection a summary: { id, kind, name, smart, item_count, covers }
 * @param {boolean} fluid fills its grid cell instead of the fixed width used in rows
 */
export default function CollectionCard({ collection, fluid = false }) {
    const { id, kind, name, smart, item_count: count = 0, covers } = collection
    const path = kind === "collection" ? `/collections/${id}` : `/playlists/${id}`
    const noun = kind === "collection" ? (count === 1 ? "item" : "items") : (count === 1 ? "track" : "tracks")

    return <li className={`group ${fluid ? "min-w-0" : "w-[8.5rem] sm:w-40 md:w-44 shrink-0"}`}>
        <Link to={path} className="block outline-none">
            <div className="relative aspect-square overflow-hidden rounded-box bg-base-300 shadow-card ring-1 ring-white/5
                transition-all duration-300 ease-out
                group-hover:-translate-y-1 group-hover:shadow-raised group-hover:ring-primary/40
                group-focus-within:ring-primary/60">
                <CollectionCover covers={covers} kind={kind} smart={smart}
                    className="size-full transition-transform duration-500 ease-out group-hover:scale-[1.04]"/>
                {smart && <span className="glass absolute top-2 left-2 rounded-full px-2 py-0.5 text-[11px] font-medium text-white">Auto</span>}
            </div>
            <div className="mt-2.5 px-0.5">
                <h3 className="text-sm font-medium leading-snug line-clamp-2 transition-colors group-hover:text-primary">{name}</h3>
                <p className="mt-0.5 text-xs text-base-content/50 truncate">{count} {noun}</p>
            </div>
        </Link>
    </li>
}

/**
 * The last card in a row of playlists or collections, starting a new one
 * @param {string} kind "playlist" or "collection"
 * @param {function} onClick opens the form for it
 */
export function NewCollectionCard({ kind, onClick, fluid = false }) {
    return <li className={`group ${fluid ? "min-w-0" : "w-[8.5rem] sm:w-40 md:w-44 shrink-0"}`}>
        <button type="button" onClick={onClick} className="block w-full text-left outline-none">
            <div className="aspect-square rounded-box border-2 border-dashed border-white/15 grid place-items-center text-base-content/50
                transition-colors group-hover:border-primary/60 group-hover:text-primary group-focus-visible:border-primary">
                <Plus size={32}/>
            </div>
            <div className="mt-2.5 px-0.5">
                <h3 className="text-sm font-medium leading-snug transition-colors group-hover:text-primary">
                    New {kind === "collection" ? "collection" : "playlist"}
                </h3>
            </div>
        </button>
    </li>
}
