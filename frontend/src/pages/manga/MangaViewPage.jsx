import { useEffect, useState} from "react";
import { NavLink, useParams} from "react-router-dom";
import api from "../../services/api";
import { mangaStatus } from "../../utils/status";
import Loading from "../../components/states/Loading";

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

    return <div className="flex flex-col gap-2">
        <img src={manga.cover_image_url}/>
        <h2>{manga.title}</h2>
        <section className="join">
            <span className={mangaStatus(manga.metadata.status)}>{manga.metadata.status}</span>
            <span>{manga.metadata.total_chapters}</span>
            <ul>
                {(manga.metadata.genres ?? []).map((genre, i) => (
                    <li key={`${manga.title}-${i}`}>{genre}</li>
                ))}
            </ul>
        </section>

        <div>
            <button className="btn btn-primary" onClick={handleRequest} disabled={requesting}>
                {requesting ? <Loading /> : "Request Download"}
            </button>
            {requestMsg && <p className="mt-2 text-sm">{requestMsg}</p>}
        </div>

        <ul className="list">
            {(manga.metadata.chapters ?? []).map(chapter => (
                <li key={`${manga.title}-${chapter.id}`} className="list-item">
                    <NavLink to={`/manga/${id}/chapters/${chapter.id}/read`}>
                        {chapter.title ?? `Chapter ${chapter.chapter_number}`}
                    </NavLink>
                </li>
            ))}
        </ul>
    </div>
}