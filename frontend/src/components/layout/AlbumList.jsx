import { Link } from "react-router-dom"
import { Play } from "lucide-react"
import useAudioStore from "../../stores/useAudioStore"
import api from "../../services/api"
import Cover from "../Cover"
import EmptyState from "../states/EmptyState"
import { CardRowSkeleton } from "../states/Skeletons"
import { toast } from "../../stores/useToastStore"

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

    if (loading) return <CardRowSkeleton heading={heading}/>

    if (!albums || albums.length === 0) {
        return <div className="my-4">
            <h2 className="font-bold mb-2">{heading}</h2>
            <EmptyState compact title={emptyText} hint={emptyHint} action={emptyAction}/>
        </div>
    }

    return <div className="my-4">
        <h2 className="font-bold mb-2">{heading}</h2>
        <ul className="flex gap-4 overflow-x-auto flex-nowrap">
            {albums.map(album => (
                <Link key={album.id} to={`/albums/${album.id}`}>
                    <li className="card border border-base-300 w-44 shrink-0 bg-base-300 h-full">
                        <figure className="relative">
                            <Cover src={album.cover_image_url} alt={album.title} className="w-full h-44 object-cover"/>
                            <button
                                className="absolute bottom-2 right-2 btn btn-circle btn-sm btn-primary"
                                onClick={(e) => handlePlay(e, album.id)}
                                aria-label={`Play ${album.title}`}
                            >
                                <Play size={14} />
                            </button>
                        </figure>
                        <div className="card-body p-2 gap-1">
                            <h3 className="card-title text-sm line-clamp-2">{album.title}</h3>
                            <span className="text-xs opacity-70">{album.artist}</span>
                            {album.track_count != null && <span className="text-xs opacity-50">{album.track_count} tracks</span>}
                        </div>
                    </li>
                </Link>
            ))}
        </ul>
        <div className="divider"></div>
    </div>
}
