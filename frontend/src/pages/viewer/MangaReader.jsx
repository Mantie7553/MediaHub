import { useRef, useEffect } from "react";
import { useParams } from "react-router-dom"
import { ChevronLeft, ChevronRight } from "lucide-react"
import { useMediaItem, usePages } from "../../hooks";
import Loading from "../../components/states/Loading";
import Error from "../../components/states/Error";
import api from "../../services/api";

export default function MangaReader() {
    const { id, chapterId } = useParams();
    const { item, loading, error } = useMediaItem(id);

    if (loading) return <Loading />
    if (error) return <Error error={error} />
    if (!item) return null

    const chapter = item.metadata?.chapters?.find(c => c.id === chapterId);
    return <MangaReaderInner id={id} chapterId={chapterId} chapter={chapter} />
}

function MangaReaderInner({ id, chapterId, chapter }) {
    const initialPage = chapter?.last_page_read ?? 0;
    const totalPages = chapter?.page_count ?? 0;
    const { currentPage, setCurrentPage, imageSrc, loading, error } = usePages(id, chapterId, initialPage);
    const pagesSinceLastSave = useRef(0);
    const currentPageRef = useRef(currentPage);

    useEffect(() => {
        currentPageRef.current = currentPage;
    }, [currentPage]);

    useEffect(() => {
        return () => saveProgress(currentPageRef.current);
    }, [chapterId]);

    function saveProgress(page) {
        const completed = page >= totalPages - 1;
        api.put(`/manga/${id}/chapters/${chapterId}/progress`, {
            last_page_read: page,
            completed,
        }).catch(() => {});
        pagesSinceLastSave.current = 0;
    }

    function changePage(newPage) {
        setCurrentPage(newPage);
        pagesSinceLastSave.current += 1;
        if (pagesSinceLastSave.current >= 5) {
            saveProgress(newPage);
        }
    }

    function handlePageUp() {
        const newPage = currentPage + 1 < totalPages ? currentPage + 1 : currentPage;
        changePage(newPage);
    }

    function handlePageDown() {
        const newPage = currentPage - 1 >= 0 ? currentPage - 1 : currentPage;
        changePage(newPage);
    }

    useEffect(() => {
        function handleKeyDown(e) {
            if (e.key === "ArrowRight") handlePageUp();
            if (e.key === "ArrowLeft") handlePageDown();
        }
        window.addEventListener("keydown", handleKeyDown);
        return () => window.removeEventListener("keydown", handleKeyDown);
    }, [totalPages, currentPage]);

    if (loading) return <Loading />
    if (error) return <Error error={error} />

    const progressPct = totalPages ? Math.round((currentPage / totalPages) * 100) : 0;

    return (
        <div className="flex flex-col items-center">
            <div className="flex-1 overflow-hidden relative">
                <img src={imageSrc} className="max-h-[calc(100vh-5rem)] max-w-full object-contain"/>
                <div className="absolute inset-y-0 left-0 w-1/2 cursor-pointer hover:bg-black/10" onClick={handlePageDown}></div>
                <div className="absolute inset-y-0 right-0 w-1/2 cursor-pointer hover:bg-black/10" onClick={handlePageUp}></div>
            </div>
            <div className="flex flex-col items-center gap-2 w-full max-w-md">
                <progress className="progress progress-primary w-full" value={progressPct} max="100"/>
                <div className="flex gap-2 items-center">
                    <button onClick={handlePageDown} disabled={currentPage === 0} className="btn"><ChevronLeft size={24} strokeWidth={4}/></button>
                    <p className={currentPage !== totalPages - 1 ? "text-neutral-content" : ""}>{currentPage + 1} / <strong>{totalPages}</strong></p>
                    <button onClick={handlePageUp} disabled={currentPage >= totalPages - 1} className="btn"><ChevronRight size={24} strokeWidth={4}/></button>
                </div>
            </div>
        </div>
    )
}