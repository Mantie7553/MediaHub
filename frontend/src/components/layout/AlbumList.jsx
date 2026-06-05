import { Link } from "react-router-dom"
import { Music, Play } from "lucide-react"
import useAudioStore from "../../stores/useAudioStore"
import api from "../../services/api"

export default function AlbumList({ albums, heading }) {

    const playAlbum = useAudioStore(state => state.playAlbum)

    function handlePlay(e, albumId) {
        e.preventDefault()
        e.stopPropagation()
        api.get(`/albums/${albumId}`).then(res => {
            const a = res.data
            const tracks = a.tracks.map(t => ({
                id: t.media_item_id,
                title: t.title,
                artist: a.artist,
                thumbnail: a.cover_image_url,
            }))
            playAlbum(tracks, 0)
        })
    }

    if (!albums || albums.length === 0) {
        return <div className="my-4">
            <h2 className="font-bold mb-2">{heading}</h2>
            <div className="flex items-center justify-center h-32 w-full border border-dashed border-base-300 rounded-lg">
                <p className="text-base-content/50 text-sm pl-2">Nothing here yet</p>
            </div>
        </div>
    }

    return <div className="my-4">
        <h2 className="font-bold mb-2">{heading}</h2>
        <ul className="flex gap-4 overflow-x-auto flex-nowrap">
            {albums.map(album => (
                <Link key={album.id} to={`/albums/${album.id}`}>
                    <li className="card border border-base-300 w-44 shrink-0 bg-base-300 h-full">
                        <figure className="relative">
                            {album.cover_image_url ? (
                                <img src={album.cover_image_url} className="w-full h-44 object-cover" />
                            ) : (
                                <div className="w-full h-44 bg-base-200 flex items-center justify-center">
                                    <Music size={32} />
                                </div>
                            )}
                            <button
                                className="absolute bottom-2 right-2 btn btn-circle btn-sm btn-primary"
                                onClick={(e) => handlePlay(e, album.id)}
                            >
                                <Play size={14} />
                            </button>
                        </figure>
                        <div className="card-body p-2 gap-1">
                            <h3 className="card-title text-sm line-clamp-2">{album.title}</h3>
                            <span className="text-xs opacity-70">{album.artist}</span>
                            <span className="text-xs opacity-50">{album.track_count} tracks</span>
                        </div>
                    </li>
                </Link>
            ))}
        </ul>
        <div className="divider"></div>
    </div>
}