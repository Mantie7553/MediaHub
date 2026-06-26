import { useParams, useNavigate } from "react-router-dom";
import Loading from "../../components/states/Loading";
import Error from "../../components/states/Error";
import { useMediaItem, useUserContent } from "../../hooks";
import Format from "../../utils/format";
import api from "../../services/api";
import { mediaStatusBadge } from "../../utils/status";
import { ArrowLeft } from "lucide-react";

export default function MovieDetailsPage() {
    const navigate = useNavigate();
    const { userContentMap, refresh } = useUserContent();
    const { id } = useParams();
    const { item: movie, loading, error } = useMediaItem(id);
    const userEntry = userContentMap[id];

    function updateStatus(status) {
        if (userEntry) {
            api.put(`/me/media/${userEntry.id}`, { status }).then(() => refresh());
        } else {
            api.post(`/me/media`, { media_item_id: id, status }).then(() => refresh());
        }
    }

    if (loading) return <Loading />
    if (error) return <Error error={error} />
    if (!movie) return null

    return (
        <div className="flex flex-col gap-8">
            <button className="btn btn-ghost btn-sm self-start" onClick={() => navigate("/")}>
                <ArrowLeft size={18} strokeWidth={2}/> Back
            </button>
            <div className="flex gap-6">
                <img src={movie.cover_image_url} className="w-48 h-64 object-cover rounded-md" />
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

                    <div className="flex gap-2">
                        {movie.metadata.file_path && (
                            <button
                                className="btn btn-primary"
                                onClick={() => navigate(`/watch/movie/${id}`)}
                            >
                                Watch
                            </button>
                        )}
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
                            </ul>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}