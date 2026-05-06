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

    function handlePageUp() {
        setCurrentPage(prev => prev + 1 < totalPages ? prev + 1 : prev)
    }

    function handlePageDown() {
        setCurrentPage(prev => prev - 1 >= 0 ? prev - 1 : prev)
    }

    // get the chapter
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

    // get the specific page
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

    if (loading) return <div className="flex justify-center p-10"><span className="loading loading-spinner loading-lg"></span></div>
    if (error) return <div className="alert alert-error">{error}</div>
    
    return <div className="flex flex-col gap-4 mx-auto">
        <img src={imageSrc} />
        <div className="flex flex-gap-2">
        <button onClick={handlePageDown} disabled={currentPage === 0} className="btn">Prev</button>
        <p className={currentPage !== totalPages ? "text-neutral-content" : ""}>{currentPage + 1} / <strong>{totalPages}</strong></p>
        <button onClick={handlePageUp} disabled={currentPage >= totalPages - 1} className="btn">Next</button>
        </div>
    </div>
}