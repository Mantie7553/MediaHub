import { useRef, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import ErrorState from "../../components/states/ErrorState";
import NotFound from "../../components/states/NotFound";
import { DetailsSkeleton } from "../../components/states/Skeletons";
import { toast } from "../../stores/useToastStore";
import { openViewer } from "../../utils/navigation";
import { useMediaItem, useUserContent, useRemoveFromList } from "../../hooks";
import DeleteFromServerButton from "../../components/DeleteFromServerButton";
import Format from "../../utils/format";
import { ListPlus, PencilLine, Play, RotateCcw, Upload } from "lucide-react";
import DetailsHero from "../../components/details/DetailsHero";
import { ActionMenu, ActionMenuItem, ListStatusMenu } from "../../components/details/Menus";
import EditDetailsModal from "../../components/modals/EditDetailsModal";
import UploadFilesModal from "../../components/modals/UploadFilesModal";
import { userIsAdmin } from "../../utils/auth";
import { saveListEntry } from "../../services/mediaService";
import useCollectionPicker from "../../hooks/useCollectionPicker";

export default function MovieDetailsPage() {
    const navigate = useNavigate();
    const { userContentMap, refresh } = useUserContent();
    const { id } = useParams();
    const { item: movie, loading, error, notFound, refetch } = useMediaItem(id);
    const picker = useCollectionPicker();
    const userEntry = userContentMap[id];
    const removeEntry = useRemoveFromList(userEntry, refresh);
    const editRef = useRef(null);
    const uploadRef = useRef(null);
    // a cover just uploaded, with a version so the browser doesn't show the one it cached
    const [newCover, setNewCover] = useState(null);
    const isAdmin = userIsAdmin();

    function updateStatus(status) {
        saveListEntry(userEntry, { media_item_id: id }, { status })
            .then(() => refresh())
            .catch(err => toast.error(err, "Could not update your list"));
    }

    if (loading) return <DetailsSkeleton />
    if (notFound) return <NotFound title="Movie not found" hint="This movie isn't in the library any more. It may have been removed." />
    if (error) return <ErrorState error={error} title="Couldn't load this movie" onRetry={refetch} showBack />
    if (!movie) return null

    // resume unless barely started or within the last 2 minutes (credits)
    const position = movie.metadata.position_secs ?? 0
    const duration = movie.metadata.duration_secs ?? 0
    const canResume = position > 30 && (!duration || position < duration - 120)

    const runtime = movie.metadata.runtime_mins
    const year = movie.release_date && Format.year(movie.release_date) !== "0" ? Format.year(movie.release_date) : null

    return (
        <div className="flex flex-col gap-8">
            <DetailsHero
                cover={{ src: newCover ?? movie.cover_image_url }}
                eyebrow="Movie"
                title={movie.title}
                meta={[
                    year,
                    runtime && (runtime >= 60 ? `${Math.floor(runtime / 60)}h ${runtime % 60}m` : `${runtime}m`),
                    movie.metadata.director && `Directed by ${movie.metadata.director}`,
                ]}
                genres={movie.metadata.genres ?? []}
                description={movie.description}
                html
                note={!movie.metadata.file_path && "Not downloaded yet, so it can't be played."}
                primary={movie.metadata.file_path && <button
                    className="btn btn-primary btn-lg shadow-glow px-6"
                    onClick={() => openViewer(navigate, `/watch/movie/${id}`, { position: canResume ? position : 0 })}
                >
                    <Play size={18} fill="currentColor"/>
                    {canResume ? `Resume from ${Format.duration(position)}` : "Play"}
                </button>}
                actions={<>
                    {movie.metadata.file_path && canResume && (
                        <button
                            className="btn btn-square sm:w-auto sm:px-4 border-white/10 bg-white/10 hover:bg-white/15"
                            onClick={() => openViewer(navigate, `/watch/movie/${id}`, { position: 0 })}
                            aria-label="Start over"
                        >
                            <RotateCcw size={18}/> <span className="hidden sm:inline">Start over</span>
                        </button>
                    )}
                    <ListStatusMenu
                        status={userEntry?.status}
                        type={movie.type}
                        options={["current", "completed", "wishlist"]}
                        onSelect={updateStatus}
                        onRemove={removeEntry}
                    />
                    <ActionMenu>
                        <ActionMenuItem icon={ListPlus} label="Add to a collection…" onClick={() => picker.open({ id, type: movie.type, title: movie.title })}/>
                        {isAdmin && <>
                            <ActionMenuItem icon={PencilLine} label="Edit details…" onClick={() => editRef.current.showModal()}/>
                            {!movie.metadata.file_path && <ActionMenuItem icon={Upload} label="Add the movie file…" onClick={() => uploadRef.current.showModal()}/>}
                            <DeleteFromServerButton asMenuItem item={{ id, title: movie.title, type: movie.type }} />
                        </>}
                    </ActionMenu>
                </>}
            />
            {canResume && duration > 0 && (
                <div className="surface flex items-center gap-4 px-5 py-4 max-w-xl animate-fade-up">
                    <div className="flex-1">
                        <p className="text-sm font-medium">Watched {Format.duration(position)} of {Format.duration(duration)}</p>
                        <div className="mt-2 h-1.5 rounded-full bg-white/10 overflow-hidden">
                            <div className="h-full rounded-full bg-primary" style={{ width: `${Math.round((position / duration) * 100)}%` }}></div>
                        </div>
                    </div>
                    <span className="text-sm text-base-content/50 tabular-nums">{Math.max(1, Math.ceil((duration - position) / 60))} min left</span>
                </div>
            )}

            {isAdmin && <>
                <EditDetailsModal dialogRef={editRef} item={movie} html onSaved={({ cover }) => { setNewCover(cover ?? null); refetch(false); }}/>
                <UploadFilesModal dialogRef={uploadRef} kind="movie" target={{ id, title: movie.title }} onAdded={() => refetch(false)}/>
            </>}
            {picker.modal}
        </div>
    )
}
