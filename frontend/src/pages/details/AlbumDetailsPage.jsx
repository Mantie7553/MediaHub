import { useRef, useState } from "react"
import { useParams } from "react-router-dom"
import { Play, BookmarkPlus, Check, PencilLine, Upload, ListPlus } from "lucide-react"
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
import { ActionMenu, ActionMenuItem } from "../../components/details/Menus"
import { getRole } from "../../utils/auth"
import Format from "../../utils/format"
import EditDetailsModal from "../../components/modals/EditDetailsModal"
import UploadFilesModal from "../../components/modals/UploadFilesModal"
import useAdminPieces from "../../hooks/useAdminPieces"
import useCollectionPicker from "../../hooks/useCollectionPicker"
import TrackList from "../../components/details/TrackList"

export default function AlbumDetailsPage() {
    const dialogRef = useRef(null);
    const editRef = useRef(null);
    const uploadRef = useRef(null);
    // a cover just uploaded, with a version so the browser doesn't show the one it cached
    const [uploadedCover, setUploadedCover] = useState(null);
    const isAdmin = getRole() === "admin";
    const { id } = useParams();
    const { userContentMap, refresh } = useUserContent();
    const userContent = userContentMap[id];
    const removeEntry = useRemoveFromList(userContent, refresh);
    const { data: album, loading, error, notFound, reload } = useApi(`/albums/${id}`);
    const playAlbum = useAudioStore(state => state.playAlbum);
    const pieces = useAdminPieces(() => reload(true));
    const picker = useCollectionPicker();

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
    const cover = uploadedCover ?? album.cover_image_url
    const queueTracks = tracks.map(t => ({
        id: t.media_item_id,
        title: t.title,
        artist: album.artist,
        thumbnail: cover,
    }))
    const totalSecs = tracks.reduce((sum, t) => sum + (t.duration_secs ?? 0), 0)

    return (
        <div className="flex flex-col gap-8">
            <DetailsHero
                cover={{ src: cover }}
                square
                eyebrow="Album"
                title={album.title}
                meta={[
                    album.artist && <span className="font-medium text-base-content">{album.artist}</span>,
                    `${tracks.length} ${tracks.length === 1 ? "track" : "tracks"}`,
                    totalSecs > 0 && `${Math.round(totalSecs / 60)} min`,
                ]}
                description={album.description}
                primary={<button className="btn btn-primary btn-lg shadow-glow px-6" onClick={() => playAlbum(queueTracks, 0, `/albums/${id}`)} disabled={tracks.length === 0}>
                    <Play size={18} fill="currentColor"/> Play
                </button>}
                actions={<>
                    <button className="btn flex-1 sm:flex-none border-white/10 bg-white/10 hover:bg-white/15 gap-2 font-medium" onClick={() => dialogRef.current.showModal()}>
                        {userContent ? <Check size={16} className="text-primary"/> : <BookmarkPlus size={16}/>}
                        {userContent ? Format.statusLabel(userContent.status, "music_album") : "Add to list"}
                    </button>
                    <ActionMenu>
                        <ActionMenuItem icon={ListPlus} label="Add to playlist or collection…" onClick={() => picker.open({ id, type: "album", title: album.title })}/>
                        {isAdmin && <>
                            <ActionMenuItem icon={PencilLine} label="Edit album…" onClick={() => editRef.current.showModal()}/>
                            <ActionMenuItem icon={Upload} label="Add tracks…" onClick={() => uploadRef.current.showModal()}/>
                            <DeleteFromServerButton asMenuItem item={{ id, title: album.title, type: "music_album" }} />
                        </>}
                    </ActionMenu>
                </>}
            />

            {/* Track List */}
            {tracks.length === 0 ? (
                <EmptyState compact title="No tracks yet" hint="Tracks show up here once they've downloaded."
                    action={isAdmin ? { label: "Add tracks", onClick: () => uploadRef.current.showModal() } : undefined} />
            ) : (
                <TrackList
                    tracks={tracks.map(t => ({ key: t.media_item_id, id: t.media_item_id, title: t.title, number: t.track_number, duration_secs: t.duration_secs }))}
                    onPlay={i => playAlbum(queueTracks, i, `/albums/${id}`)}
                    menu={(track, i) => pieces.menu({
                        kind: "track", id: track.id, label: track.title,
                        title: track.title, artist: tracks[i].artist || album.artist, track_number: tracks[i].track_number,
                    }, <ActionMenuItem icon={ListPlus} label="Add to playlist…" onClick={() => picker.open({ id: track.id, type: "music_track", title: track.title })}/>)}
                />
            )}

            {isAdmin && <>
                <EditDetailsModal
                    dialogRef={editRef}
                    item={{ ...album, type: "music_album", cover_image_url: cover }}
                    onSaved={({ cover }) => { if (cover) setUploadedCover(cover); reload(true); }}
                />
                <UploadFilesModal
                    dialogRef={uploadRef}
                    kind="tracks"
                    target={{ id, title: album.title }}
                    have={tracks.map(t => t.title.trim().toLowerCase())}
                    onAdded={() => reload(true)}
                />
            </>}
            {pieces.modals}
            {picker.modal}

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
