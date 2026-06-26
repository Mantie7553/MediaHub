import { useEffect, useState, useRef } from "react"
import { useNavigate, useParams } from "react-router-dom"
import { Play, Clock, Music, ArrowLeft, BookmarkPlus } from "lucide-react"
import api from "../../services/api"
import Loading from "../../components/states/Loading"
import Error from "../../components/states/Error"
import useAudioStore from "../../stores/useAudioStore"
import { useUserContent } from "../../hooks"
import AddToListModal from "../../components/modals/AddToListModal"

export default function AlbumDetailsPage() {
    const navigate = useNavigate();
    const dialogRef = useRef(null);
    const { id } = useParams();
    const { userContentMap, refresh } = useUserContent();
    const userContent = userContentMap[id];
    const [album, setAlbum] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");
    const playAlbum = useAudioStore(state => state.playAlbum);

    useEffect(() => {
        api.get(`/albums/${id}`)
            .then(res => setAlbum(res.data))
            .catch(err => setError(err.message))
            .finally(() => setLoading(false))
    }, [id])

    function formatDuration(secs) {
        if (!secs) return "--:--"
        const m = Math.floor(secs / 60)
        const s = secs % 60
        return `${m}:${String(s).padStart(2, "0")}`
    }

    function handleAddToList({ status, score }) {
       const payload = { album_id: id, status, rating: score || null }
       if (userContent) {
           api.put(`/me/media/${userContent.id}`, { status, rating: score || null }).then(() => refresh())
       } else {
           api.post("/me/media", payload).then(() => refresh())
       }
   }

    if (loading) return <Loading />
    if (error) return <Error error={error} />
    if (!album) return null

    const queueTracks = album.tracks.map(t => ({
        id: t.media_item_id,
        title: t.title,
        artist: album.artist,
        thumbnail: album.cover_image_url,
    }))

    return (
        <div className="flex flex-col gap-8">
            <button className="btn btn-ghost btn-sm self-start" onClick={() => navigate("/")}>
                <ArrowLeft size={18} strokeWidth={2}/> Back
            </button>
            {/* Album Header */}
            <div className="flex gap-6">
                {album.cover_image_url ? (
                    <img src={album.cover_image_url} className="w-48 h-48 object-cover rounded-md" />
                ) : (
                    <div className="w-48 h-48 rounded-md bg-base-300 flex items-center justify-center">
                        <Music size={48} />
                    </div>
                )}
                <div className="flex flex-col gap-2 justify-end">
                    <p className="text-sm opacity-70">Album</p>
                    <h2 className="text-3xl font-bold">{album.title}</h2>
                    <p className="opacity-70">{album.artist} · {album.tracks.length} tracks</p>
                    <button className="btn btn-primary btn-sm w-fit" onClick={() => playAlbum(queueTracks, 0)}>
                        <Play size={16} /> Play
                    </button>
                    <button className="btn btn-outline btn-sm w-fit" onClick={() => dialogRef.current.showModal()}>
                        <BookmarkPlus size={16} /> {userContent ? "Edit List" : "Add to List"}
                    </button>
                </div>
            </div>

            {/* Track List */}
            <div className="flex flex-col">
                <div className="flex items-center gap-3 px-3 py-2 text-xs opacity-50 border-b border-base-300">
                    <span className="w-8 text-right">#</span>
                    <span className="flex-1">Title</span>
                    <Clock size={12} />
                </div>
                {album.tracks.map((track, i) => (
                    <div
                        key={track.media_item_id}
                        className="flex items-center gap-3 px-3 py-3 hover:bg-base-200 rounded-lg cursor-pointer group"
                        onClick={() => playAlbum(queueTracks, i)}
                    >
                        <span className="w-8 text-right text-sm opacity-50 group-hover:hidden">
                            {track.track_number ?? i + 1}
                        </span>
                        <span className="w-8 text-right hidden group-hover:block text-primary">
                            <Play size={14} />
                        </span>
                        <span className="flex-1 text-sm font-medium">{track.title}</span>
                        <span className="text-xs opacity-50">{formatDuration(track.duration_secs)}</span>
                    </div>
                ))}
            </div>

            <AddToListModal
                dialogRef={dialogRef}
                item={{ title: album.title, type: "music_album" }}
                initialValues={userContent}
                onConfirm={handleAddToList}
            />
        </div>
    )
}