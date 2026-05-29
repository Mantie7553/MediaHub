import { useEffect, useState} from "react";
import { NavLink, useParams} from "react-router-dom";
import api from "../../services/api";
import Loading from "../../components/states/Loading";
import Error from "../../components/states/Error";
import { useMediaItem, useCreateRequest } from "../../hooks";
import { animeBadge } from "../../utils/status";

export default function AnimeViewPage() {
    const { id } = useParams();
    const { item: anime, loading, error } = useMediaItem(id);
    const { requesting, requestMsg, createRequest } = useCreateRequest(id);

    if (loading) return <Loading />
    if (error) return <Error error={error} />
    if (!anime) return null

    return <div className="flex flex-col">
            <div className="flex gap-6">
                <img src={anime.cover_image_url} className="w-48 h-64 object-cover rounded-md"/>
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
        </div>
}