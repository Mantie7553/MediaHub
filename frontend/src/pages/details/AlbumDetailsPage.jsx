import { useRef } from "react"
import { useParams } from "react-router-dom"
import { Play, Clock, BookmarkPlus, Check } from "lucide-react"
import api from "../../services/api"
import useApi from "../../hooks/useApi"
import ErrorState from "../../components/states/ErrorState"
import NotFound from "../../components/states/NotFound"
import EmptyState from "../../components/states/EmptyState"
import { DetailsSkeleton } from "../../components/states/Skeletons"
import { toast } from "../../stores/useToastStore"
import DeleteFromServerButton from "../../components/DeleteFromServerButton"
import useAudioStore from "../../stores/useAudioStore"
import { useUserContent, useRemoveFromList } from "../../hooks"
import AddToListModal from "../../components/modals/AddToListModal"
import DetailsHero from "../../components/details/DetailsHero"
import { ActionMenu } from "../../components/details/Menus"
import { getRole } from "../../utils/auth"
import Format from "../../utils/format"

export default function AlbumDetailsPage() {
    const dialogRef = useRef(null);
    const { id } = useParams();
    const { userContentMap, refresh } = useUserContent();
    const userContent = userContentMap[id];
    const removeEntry = useRemoveFromList(userContent, refresh);
    const { data: album, loading, error, notFound, reload } = useApi(`/albums/${id}`);
    const playAlbum = useAudioStore(state => state.playAlbum);
    const currentTrack = useAudioStore(state => state.currentTrack);
    const isPlaying = useAudioStore(state => state.isPlaying);

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

    if (loading) return <DetailsSkeleton square />
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
    const totalSecs = tracks.reduce((sum, t) => sum + (t.duration_secs ?? 0), 0)

    return (
        <div className="flex flex-col gap-8">
            <DetailsHero
                cover={{ src: album.cover_image_url }}
                square
                eyebrow="Album"
                title={album.title}
                meta={[
                    album.artist && <span className="font-medium text-base-content">{album.artist}</span>,
                    `${tracks.length} ${tracks.length === 1 ? "track" : "tracks"}`,
                    totalSecs > 0 && `${Math.round(totalSecs / 60)} min`,
                ]}
                primary={<button className="btn btn-primary btn-lg shadow-glow px-6" onClick={() => playAlbum(queueTracks, 0)} disabled={tracks.length === 0}>
                    <Play size={18} fill="currentColor"/> Play
                </button>}
                actions={<>
                    <button className="btn flex-1 sm:flex-none border-white/10 bg-white/10 hover:bg-white/15 gap-2 font-medium" onClick={() => dialogRef.current.showModal()}>
                        {userContent ? <Check size={16} className="text-primary"/> : <BookmarkPlus size={16}/>}
                        {userContent ? Format.statusLabel(userContent.status, "music_album") : "Add to list"}
                    </button>
                    {getRole() === "admin" && <ActionMenu>
                        <DeleteFromServerButton asMenuItem item={{ id, title: album.title, type: "music_album" }} />
                    </ActionMenu>}
                </>}
            />

            {/* Track List */}
            {tracks.length === 0 ? (
                <EmptyState compact title="No tracks yet" hint="Tracks show up here once they've downloaded." />
            ) : (
                <section className="surface p-1.5 sm:p-2 animate-fade-up">
                    <div className="flex items-center gap-4 px-3 py-2 text-xs uppercase tracking-wider text-base-content/40 border-b border-white/5 mb-1">
                        <span className="w-8 text-right">#</span>
                        <span className="flex-1">Title</span>
                        <Clock size={14} />
                    </div>
                    <ol className="flex flex-col gap-0.5">
                        {tracks.map((track, i) => {
                            const playing = currentTrack?.id === track.media_item_id
                            return <li key={track.media_item_id}>
                                <button
                                    className={`group w-full flex items-center gap-4 px-3 py-2.5 rounded-field text-left transition-colors hover:bg-white/5
                                        ${playing ? "bg-primary/8" : ""}`}
                                    onClick={() => playAlbum(queueTracks, i)}
                                    aria-label={`Play ${track.title}`}
                                >
                                    <span className="w-8 flex justify-end text-sm tabular-nums text-base-content/40">
                                        {playing
                                            ? <Equalizer animate={isPlaying}/>
                                            : <>
                                                <span className="group-hover:hidden">{track.track_number ?? i + 1}</span>
                                                <Play size={14} fill="currentColor" className="hidden group-hover:block text-primary"/>
                                            </>}
                                    </span>
                                    <span className={`flex-1 min-w-0 truncate text-sm font-medium ${playing ? "text-primary" : ""}`}>{track.title}</span>
                                    <span className="text-xs tabular-nums text-base-content/45">{formatDuration(track.duration_secs)}</span>
                                </button>
                            </li>
                        })}
                    </ol>
                </section>
            )}

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

/**
 * Three bouncing bars marking the track that's playing
 * @param {boolean} animate bounces while playing, still when paused
 */
function Equalizer({ animate }) {
    return <span className="flex items-end gap-[2px] h-3.5 text-primary" aria-label="Now playing">
        {[0, 0.2, 0.4].map(delay => (
            <span key={delay} className={`w-[3px] rounded-sm bg-current ${animate ? "animate-eq" : "h-1.5"}`} style={{ animationDelay: `${delay}s` }}></span>
        ))}
    </span>
}
