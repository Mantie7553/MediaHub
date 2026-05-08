import { useEffect, useState} from "react";
import { NavLink, useParams} from "react-router-dom";
import api from "../../services/api";
import { mangaBadge } from "../../utils/status";
import Loading from "../../components/states/Loading";
import Error from "../../components/states/Error"

/**
 * Manga view page layout
 * @returns
 */
export default function DisplayPage() {
    const [manga, setManga] = useState(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState("");
    const [requesting, setRequesting] = useState(false);
    const [requestMsg, setRequestMsg] = useState("");
    const { id } = useParams()

    useEffect(() => {
        setLoading(true);
        api.get(`/media/${id}`)
        .then(resp => setManga(resp.data))
        .catch(err => setError(err.message))
        .finally(() => setLoading(false))
    }, [])

    function handleRequest() {
        setRequesting(true);
        setRequestMsg("");
        api.post("/requests", { media_item_id: id })
        .then(() => setRequestMsg("Download requested!"))
        .catch(err => setRequestMsg(err.response?.data?.error ?? err.message))
        .finally(() => setRequesting(false))
    }

    if (loading) return <Loading />
    if (error) return <Error error={error} />
    if (!manga) return null

    return <div className="flex flex-col">
        <div className="flex gap-6">
            <img src={manga.cover_image_url} className="w-48 h-64 object-cover rounded-md"/>
            <div className="flex flex-col gap-3">
                <h2 className="text-2xl font-bold">{manga.title}</h2>
                <span className={`badge ${mangaBadge(manga.metadata.status)}`}>{manga.metadata.status}</span>
                <span className="text-sm text-neutral-content">{manga.metadata.total_chapters ?? "N/A"} chapters</span>
                <div className="flex flex-wrap gap-1">
                    {(manga.metadata.genres ?? []).map((genre, i) => (
                        <span key={`${manga.title}-${i}`} className="badge">{genre}</span>
                    ))}
                </div>
                <div>
                    <button className="btn btn-primary" onClick={handleRequest} disabled={requesting}>
                        {requesting ? <Loading /> : "Request Download"}
                    </button>
                    {requestMsg && <p className="mt-2 text-sm">{requestMsg}</p>}
                </div>
            </div>
        </div>

        <h3 className="font-bold text-lg mt-4">Chapters</h3>
        <ul className="list">
            {(manga.metadata.chapters ?? []).map(chapter => (
                <li key={`${manga.title}-${chapter.id}`} className="list-item hover:bg-base-300 transition-colors px-2 py-1">
                    <NavLink to={`/manga/${id}/chapters/${chapter.id}/read`} className="block w-full">
                        {chapter.title ?? `Chapter ${chapter.chapter_number}`}
                    </NavLink>
                </li>
            ))}
        </ul>
    </div>
}