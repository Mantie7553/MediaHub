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
    const { id } = useParams()

    useEffect(() => {
        setLoading(true);
        api.get(`/media/${id}`)
        .then(resp => setManga(resp.data))
        .catch(err => setError(err.message))
        .finally(() => setLoading(false))
    }, [])

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
        <ul>
            {(manga.metadata.chapters ?? []).map(chapter => (
                <li key={`${manga.title}-${chapter.id}`}>
                    <NavLink to={`/manga/${id}/chapters/${chapter.id}/read`} >
                        {chapter.title}
                    </NavLink>
                </li>
            ))}
        </ul>
    </div>
}