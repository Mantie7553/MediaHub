import { Link } from "react-router-dom"
import { Play } from "lucide-react"
import useAudioStore from "../../stores/useAudioStore"
import api from "../../services/api"
import Cover from "../Cover"
import EmptyState from "../states/EmptyState"
import { CardRowSkeleton } from "../states/Skeletons"
import { toast } from "../../stores/useToastStore"
import SectionHeader from "./SectionHeader"
import Shelf from "./Shelf"

/**
 * A row of album cards with a play button on each
 * @param {any[]} albums the albums to show
 * @param {string} heading the heading for this list
 * @param {boolean} loading shows placeholder cards instead of the albums
 * @param {string} emptyText what to say when there are no albums
 * @param {string} emptyHint what the user can do about it
 * @param {object} emptyAction optional { label, to } shown when there are no albums
 */
export default function AlbumList({ albums, heading, loading = false, emptyText = "Nothing here yet", emptyHint, emptyAction }) {

    const playAlbum = useAudioStore(state => state.playAlbum)

    function handlePlay(e, albumId) {
        e.preventDefault()
        e.stopPropagation()
        api.get(`/albums/${albumId}`).then(res => {
            const a = res.data
            const tracks = (a.tracks ?? []).map(t => ({
                id: t.media_item_id,
                title: t.title,
                artist: a.artist,
                thumbnail: a.cover_image_url,
            }))
            if (tracks.length === 0) {
                toast.error("This album has no tracks to play")
                return
            }
            playAlbum(tracks, 0)
        }).catch(err => toast.error(err, "Could not play this album"))
    }

    if (loading) return <CardRowSkeleton heading={heading} square/>

    if (!albums || albums.length === 0) {
        return <section>
            <SectionHeader title={heading}/>
            <EmptyState compact title={emptyText} hint={emptyHint} action={emptyAction}/>
        </section>
    }

    return <section className="animate-fade-up">
        <SectionHeader title={heading} count={albums.length}/>
        <Shelf>
            {albums.map(album => (
                <li key={album.id} className="group w-[8.5rem] sm:w-40 md:w-44 shrink-0">
                    <Link to={`/albums/${album.id}`} className="block outline-none">
                        <div className="relative aspect-square overflow-hidden rounded-box bg-base-300 shadow-card ring-1 ring-white/5
                            transition-all duration-300 ease-out
                            group-hover:-translate-y-1 group-hover:shadow-raised group-hover:ring-primary/40
                            group-focus-within:ring-primary/60">
                            <Cover src={album.cover_image_url} alt={album.title}
                                className="size-full object-cover transition-transform duration-500 ease-out group-hover:scale-[1.04]"/>
                            <div className="absolute inset-x-0 bottom-0 h-1/2 bg-linear-to-t from-black/60 to-transparent opacity-0 transition-opacity duration-300 group-hover:opacity-100 pointer-events-none"></div>
                            <button
                                className="absolute bottom-3 right-3 grid size-11 place-items-center rounded-full bg-primary text-primary-content shadow-glow
                                    opacity-0 translate-y-2 transition-all duration-300 group-hover:opacity-100 group-hover:translate-y-0 group-focus-within:opacity-100 group-focus-within:translate-y-0
                                    hover:scale-110 hover-reveal"
                                onClick={(e) => handlePlay(e, album.id)}
                                aria-label={`Play ${album.title}`}
                            >
                                <Play size={18} fill="currentColor" className="ml-0.5"/>
                            </button>
                        </div>
                        <div className="mt-2.5 px-0.5">
                            <h3 className="text-sm font-medium leading-snug line-clamp-2 transition-colors group-hover:text-primary">{album.title}</h3>
                            <p className="mt-0.5 text-xs text-base-content/50 truncate">
                                {[album.artist, album.track_count != null ? `${album.track_count} tracks` : null].filter(Boolean).join(" · ")}
                            </p>
                        </div>
                    </Link>
                </li>
            ))}
        </Shelf>
    </section>
}
