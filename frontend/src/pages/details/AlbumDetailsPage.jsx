import { useRef } from "react"
import { useNavigate, useParams } from "react-router-dom"
import { Play, Clock, Music, ArrowLeft, BookmarkPlus } from "lucide-react"
import api from "../../services/api"
import useApi from "../../hooks/useApi"
import ErrorState from "../../components/states/ErrorState"
import NotFound from "../../components/states/NotFound"
import EmptyState from "../../components/states/EmptyState"
import { DetailsSkeleton } from "../../components/states/Skeletons"
import Cover from "../../components/Cover"
import { toast } from "../../stores/useToastStore"
import { goBack } from "../../utils/navigation"
import DeleteFromServerButton from "../../components/DeleteFromServerButton"
import useAudioStore from "../../stores/useAudioStore"
import { useUserContent, useRemoveFromList } from "../../hooks"
import AddToListModal from "../../components/modals/AddToListModal"

export default function AlbumDetailsPage() {
    const navigate = useNavigate();
    const dialogRef = useRef(null);
    const { id } = useParams();
    const { userContentMap, refresh } = useUserContent();
    const userContent = userContentMap[id];
    const removeEntry = useRemoveFromList(userContent, refresh);
    const { data: album, loading, error, notFound, reload } = useApi(`/albums/${id}`);
    const playAlbum = useAudioStore(state => state.playAlbum);
    const currentTrack = useAudioStore(state => state.currentTrack);

    function formatDuration(secs) {
        if (!secs) return "--:--"
        const m = Math.floor(secs / 60)
        const s = secs % 60
        return `${m}:${String(s).padStart(2, "0")}`
    }

    function handleAddToList({ status, score }) {
       const request = userContent
           ? api.put(`/me/media/${userContent.id}`, { status, rating: score || null })
           : api.post("/me/media", { album_id: id, status, rating: score || null })
       request
           .then(() => { toast.success(userContent ? "List updated" : "Added to your list"); refresh() })
           .catch(err => toast.error(err, "Could not update your list"))
   }

    if (loading) return <DetailsSkeleton />
    if (notFound) return <NotFound title="Album not found" hint="This album isn't in the library any more. It may have been removed." />
    if (error) return <ErrorState error={error} title="Couldn't load this album" onRetry={reload} showBack />
    if (!album) return null

    const tracks = album.tracks ?? []
    const queueTracks = tracks.map(t => ({
        id: t.media_item_id,
        title: t.title,
        artist: album.artist,
        thumbnail: album.cover_image_url,
    }))

    return (
        <div className="flex flex-col gap-8">
            <button className="btn btn-ghost btn-sm self-start" onClick={() => goBack(navigate)}>
                <ArrowLeft size={18} strokeWidth={2}/> Back
            </button>
            {/* Album Header */}
            <div className="flex gap-6">
                <Cover src={album.cover_image_url} alt={album.title} className="w-48 h-48 shrink-0 object-cover rounded-md" />
                <div className="flex flex-col gap-2 justify-end">
                    <p className="text-sm opacity-70">Album</p>
                    <h2 className="text-3xl font-bold">{album.title}</h2>
                    <p className="opacity-70">{album.artist} · {tracks.length} tracks</p>
                    <button className="btn btn-primary btn-sm w-fit" onClick={() => playAlbum(queueTracks, 0)} disabled={tracks.length === 0}>
                        <Play size={16} /> Play
                    </button>
                    <button className="btn btn-outline btn-sm w-fit" onClick={() => dialogRef.current.showModal()}>
                        <BookmarkPlus size={16} /> {userContent ? "Edit List" : "Add to List"}
                    </button>
                    <DeleteFromServerButton item={{ id, title: album.title, type: "music_album" }} />
                </div>
            </div>

            {/* Track List */}
            <div className="flex flex-col">
                <div className="flex items-center gap-3 px-3 py-2 text-xs opacity-50 border-b border-base-300">
                    <span className="w-8 text-right">#</span>
                    <span className="flex-1">Title</span>
                    <Clock size={12} />
                </div>
                {tracks.length === 0 && (
                    <div className="mt-2">
                        <EmptyState compact title="No tracks yet" hint="Tracks show up here once they've downloaded." />
                    </div>
                )}
                {tracks.map((track, i) => (
                    <div
                        key={track.media_item_id}
                        className="flex items-center gap-3 px-3 py-3 hover:bg-base-200 rounded-lg cursor-pointer group"
                        onClick={() => playAlbum(queueTracks, i)}
                    >
                        <span className="w-8 text-right text-sm opacity-50 group-hover:hidden">
                            {currentTrack?.id === track.media_item_id 
                                ? <Music size={14} className="text-primary ml-auto" />
                                : track.track_number ?? i + 1}
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
                onRemove={userContent ? removeEntry : undefined}
            />
        </div>
    )
}