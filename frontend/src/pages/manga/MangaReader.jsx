import { useEffect, useState } from "react";
import { useParams } from "react-router-dom"
import api from "../../services/api";

export default function MangaReader() {
    const { id, chapterId } = useParams();
    const [imageSrc, setImageSrc] = useState("");
    const [currentPage, setCurrentPage] = useState(0);
    const [totalPages, setTotalPages] = useState(0);
    const [chapter, setChapter] = useState(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState("");

    useEffect(() => {
        setLoading(true)
        api.get(`/media/${id}`)
        .then(resp => {
            setChapter(resp.data);
            const match = resp.data.metadata.chapters?.find(c => c.id === chapterId);
            if (match) setTotalPages(match.page_count);

        })
        .catch(err => setError(err.message))
        .finally(() => setLoading(false))
    }, [])

    useEffect(() => {
        let objectUrl = null;
        api.get(`/manga/${id}/chapters/${chapterId}/pages/${currentPage}`, { responseType: 'blob' })
        .then(resp => {
            objectUrl = URL.createObjectURL(resp.data);
            setImageSrc(objectUrl);
        })
        .catch(err => setError(err.message));
        return () => { if (objectUrl) URL.revokeObjectURL(objectUrl) }
    }, [currentPage])


    return <div>
        <img src={imageSrc} />
        <button onClick={() => setCurrentPage(p => p - 1)} disabled={currentPage === 0}>Prev</button>
        <button onClick={() => setCurrentPage(p => p + 1)} disabled={currentPage >= totalPages - 1}>Next</button>
    </div>
}