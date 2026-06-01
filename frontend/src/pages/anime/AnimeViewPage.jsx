import { useEffect, useState } from "react";
import { NavLink, useParams, useNavigate } from "react-router-dom";
import api from "../../services/api";
import Loading from "../../components/states/Loading";
import Error from "../../components/states/Error";
import { useMediaItem, useCreateRequest } from "../../hooks";
import { animeBadge } from "../../utils/status";

export default function AnimeViewPage() {
    const { id } = useParams();
    const navigate = useNavigate();
    const { item: anime, loading, error } = useMediaItem(id);
    const { requesting, requestMsg, createRequest } = useCreateRequest(id);
    const [episodes, setEpisodes] = useState([]);

    useEffect(() => {
        api.get(`/media/${id}/episodes`)
            .then(res => setEpisodes(res.data ?? []))
            .catch(() => {})
    }, [id])

    if (loading) return <Loading />
    if (error) return <Error error={error} />
    if (!anime) return null

    // group episodes by season
    const seasons = episodes.reduce((acc, ep) => {
        const s = ep.season_number
        if (!acc[s]) acc[s] = []
        acc[s].push(ep)
        return acc
    }, {})

    return (
        <div className="flex flex-col gap-8">
            <div className="flex gap-6">
                <img src={anime.cover_image_url} className="w-48 h-64 object-cover rounded-md" />
                <div className="flex flex-col gap-3">
                    <h2 className="text-2xl font-bold">{anime.title}</h2>
                    <span className={`badge ${animeBadge(anime.metadata.status)}`}>{anime.metadata.status}</span>
                    <div className="flex flex-wrap gap-1">
                        {(anime.metadata.genres ?? []).map((genre, i) => (
                            <span key={`${anime.title}-${i}`} className="badge">{genre}</span>
                        ))}
                    </div>
                    <div>
                        <button className="btn btn-primary" onClick={createRequest} disabled={requesting}>
                            {requesting ? <Loading /> : "Request Download"}
                        </button>
                        {requestMsg && <p className="mt-2 text-sm">{requestMsg}</p>}
                    </div>
                </div>
            </div>

            {Object.keys(seasons).length > 0 && (
                <div className="flex flex-col gap-6">
                    {Object.entries(seasons).map(([seasonNum, eps]) => (
                        <div key={seasonNum}>
                            <h3 className="text-lg font-semibold mb-3">Season {seasonNum}</h3>
                            <div className="flex flex-col gap-2">
                                {eps.map(ep => (
                                    <div key={ep.id} className="flex items-center justify-between p-3 rounded-lg bg-base-200">
                                        <span className="text-sm">
                                            EP {ep.episode_number} — {ep.title ?? "Untitled"}
                                        </span>
                                        <button
                                            className="btn btn-sm btn-primary"
                                            onClick={() => navigate(`/watch/episode/${ep.id}`)}
                                        >
                                            Watch
                                        </button>
                                    </div>
                                ))}
                            </div>
                        </div>
                    ))}
                </div>
            )}
        </div>
    )
}