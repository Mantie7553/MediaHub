import { useParams, useNavigate } from "react-router-dom";
import ErrorState from "../../components/states/ErrorState";
import NotFound from "../../components/states/NotFound";
import { DetailsSkeleton } from "../../components/states/Skeletons";
import Cover from "../../components/Cover";
import { toast } from "../../stores/useToastStore";
import { goBack, openViewer } from "../../utils/navigation";
import { useMediaItem, useUserContent, useRemoveFromList } from "../../hooks";
import DeleteFromServerButton from "../../components/DeleteFromServerButton";
import Format from "../../utils/format";
import api from "../../services/api";
import { mediaStatusBadge } from "../../utils/status";
import { ArrowLeft } from "lucide-react";

export default function MovieDetailsPage() {
    const navigate = useNavigate();
    const { userContentMap, refresh } = useUserContent();
    const { id } = useParams();
    const { item: movie, loading, error, notFound, refetch } = useMediaItem(id);
    const userEntry = userContentMap[id];
    const removeEntry = useRemoveFromList(userEntry, refresh);

    function updateStatus(status) {
        const request = userEntry
            ? api.put(`/me/media/${userEntry.id}`, { status })
            : api.post(`/me/media`, { media_item_id: id, status });
        request
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

    return (
        <div className="flex flex-col gap-8">
            <button className="btn btn-ghost btn-sm self-start" onClick={() => goBack(navigate)}>
                <ArrowLeft size={18} strokeWidth={2}/> Back
            </button>
            <div className="flex gap-6">
                <Cover src={movie.cover_image_url} alt={movie.title} className="w-48 h-64 shrink-0 object-cover rounded-md" />
                <div className="flex flex-col gap-3">
                    <h2 className="text-2xl font-bold">{movie.title}</h2>

                    {movie.release_date && Format.year(movie.release_date) !== "0" && (
                        <span className="text-sm text-neutral-content">{Format.year(movie.release_date)}</span>
                    )}

                    {movie.metadata.runtime_mins && (
                        <span className="text-sm text-neutral-content">{movie.metadata.runtime_mins} min</span>
                    )}

                    {movie.metadata.director && (
                        <span className="text-sm text-neutral-content">Directed by {movie.metadata.director}</span>
                    )}

                    <div className="flex flex-wrap gap-1">
                        {(movie.metadata.genres ?? []).map((genre, i) => (
                            <span key={`${movie.title}-${i}`} className="badge">{genre}</span>
                        ))}
                    </div>

                    {movie.description && (
                        <p className="text-sm max-w-xl"
                        dangerouslySetInnerHTML={{__html: movie.description}}
                        />
                    )}

                    {!movie.metadata.file_path && (
                        <p className="text-sm text-neutral-content">Not downloaded yet, so it can't be played.</p>
                    )}

                    <div className="flex flex-wrap items-center gap-2">
                        {movie.metadata.file_path && <>
                            <button
                                className="btn btn-primary"
                                onClick={() => openViewer(navigate, `/watch/movie/${id}`, { position: canResume ? position : 0 })}
                            >
                                {canResume ? `Resume from ${Format.duration(position)}` : "Watch"}
                            </button>
                            {canResume && (
                                <button
                                    className="btn btn-ghost"
                                    onClick={() => openViewer(navigate, `/watch/movie/${id}`, { position: 0 })}
                                >
                                    Start over
                                </button>
                            )}
                        </>}
                        <div className="dropdown">
                            <div tabIndex={0} className={`badge ${mediaStatusBadge(userEntry?.status)} cursor-pointer`}>
                                {Format.statusLabel(userEntry?.status, movie.type)}
                            </div>
                            <ul tabIndex={0} className="dropdown-content menu bg-base-200 rounded-box z-10 p-2 shadow gap-1">
                                {["current", "completed", "wishlist"].map(option => (
                                    <li key={option}>
                                        <button onClick={() => { updateStatus(option); document.activeElement.blur(); }}>
                                            {Format.statusLabel(option, movie.type)}
                                        </button>
                                    </li>
                                ))}
                                {userEntry && (
                                    <li className="border-t border-base-300 mt-1 pt-1">
                                        <button className="text-error" onClick={() => { removeEntry(); document.activeElement.blur(); }}>
                                            Remove from list
                                        </button>
                                    </li>
                                )}
                            </ul>
                        </div>
                        <DeleteFromServerButton item={{ id, title: movie.title, type: movie.type }} />
                    </div>
                </div>
            </div>
        </div>
    )
}