import { useEffect, useState} from "react";
import { NavLink, useParams} from "react-router-dom";
import api from "../../services/api";

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

    if (loading) return <div className="flex justify-center p-10"><span className="loading loading-spinner loading-lg"></span></div>
    if (error) return <div className="alert alert-error">{error}</div>
    if (!manga) return null

    return <div>
        <img src={manga.cover_image_url}/>
        <h2>{manga.title}</h2>
        <section>
            <span>{manga.metadata.status}</span>
            <span>{manga.metadata.total_chapters}</span>
            <ul>
                {(manga.metadata.genres ?? []).map((genre, i) => (
                    <li key={`${manga.title}-${i}`}>{genre}</li>
                ))}
            </ul>
        </section>

        <div>
            <button className="btn btn-primary" onClick={handleRequest} disabled={requesting}>
                {requesting ? <span className="loading loading-spinner loading-sm"></span> : "Request Download"}
            </button>
            {requestMsg && <p className="mt-2 text-sm">{requestMsg}</p>}
        </div>

        <ul>
            {(manga.metadata.chapters ?? []).map(chapter => (
                <li key={`${manga.title}-${chapter.id}`}>
                    <NavLink to={`/manga/${id}/chapters/${chapter.id}/read`}>
                        {chapter.title ?? `Chapter ${chapter.chapter_number}`}
                    </NavLink>
                </li>
            ))}
        </ul>
    </div>
}