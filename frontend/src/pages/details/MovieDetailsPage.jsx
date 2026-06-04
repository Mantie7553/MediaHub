import { useParams, useNavigate } from "react-router-dom";
import Loading from "../../components/states/Loading";
import Error from "../../components/states/Error";
import { useMediaItem, useCreateRequest } from "../../hooks";
import Format from "../../utils/format";

export default function MovieDetailsPage() {
    const { id } = useParams();
    const navigate = useNavigate();
    const { item: movie, loading, error } = useMediaItem(id);
    const { requesting, requestMsg, createRequest } = useCreateRequest(id);

    if (loading) return <Loading />
    if (error) return <Error error={error} />
    if (!movie) return null

    return (
        <div className="flex flex-col gap-8">
            <div className="flex gap-6">
                <img src={movie.cover_image_url} className="w-48 h-64 object-cover rounded-md" />
                <div className="flex flex-col gap-3">
                    <h2 className="text-2xl font-bold">{movie.title}</h2>

                    {movie.release_date && (
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
                        <p className="text-sm max-w-xl">{movie.description}</p>
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
                        <button
                            className="btn btn-outline btn-primary"
                            onClick={createRequest}
                            disabled={requesting}
                        >
                            {requesting ? <Loading /> : "Request Download"}
                        </button>
                    </div>
                    {requestMsg && <p className="mt-2 text-sm">{requestMsg}</p>}
                </div>
            </div>
        </div>
    )
}