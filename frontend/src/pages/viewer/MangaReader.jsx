import { useRef, useEffect } from "react";
import { NavLink, useNavigate, useParams } from "react-router-dom"
import { ArrowLeft, ChevronLeft, ChevronRight } from "lucide-react"
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

    const chapters = item.metadata?.chapters ?? [];
    const chapter = chapters.find(c => c.id === chapterId);
    return <MangaReaderInner key={chapterId} id={id} chapterId={chapterId} chapter={chapter} chapters={chapters} />
}

function MangaReaderInner({ id, chapterId, chapter, chapters }) {
    const navigate = useNavigate();
    const initialPage = chapter?.last_page_read ?? 0;
    const totalPages = chapter?.page_count ?? 0;
    const { currentPage, setCurrentPage, imageSrc, loading, error } = usePages(id, chapterId, initialPage);
    const pagesSinceLastSave = useRef(0);
    const currentPageRef = useRef(currentPage);
    const currentIndex = chapters.findIndex(c => c.id === chapterId);
    const nextChapter = chapters[currentIndex + 1] ?? null;
    const prevChapter = chapters[currentIndex - 1] ?? null;

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
        if (completed) {
            api.put(`/manga/chapters/${chapterId}/read`, { read: true })
        }
    }

    function changePage(newPage) {
        setCurrentPage(newPage);
        pagesSinceLastSave.current += 1;
        if (pagesSinceLastSave.current >= 5) {
            saveProgress(newPage);
        }
    }

    function handlePageUp() {
        if (currentPage + 1 < totalPages) {
            changePage(currentPage + 1);
        } else if (nextChapter) {
            saveProgress(currentPage);
            navigate(`/manga/${id}/chapters/${nextChapter.id}/read`);
        }
    }

    function handlePageDown() {
        if (currentPage - 1 >= 0) {
            changePage(currentPage - 1);
        } else if (prevChapter) {
            saveProgress(currentPage);
            navigate(`/manga/${id}/chapters/${prevChapter.id}/read`);
        }
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

    const progressPct = totalPages ? Math.round(((currentPage + 1) / totalPages) * 100) : 0;

    return (
        <div className="flex flex-col">
            <div className="sticky top-0 z-10 bg-base-200 border-b border-base-300 w-full">
                <div className="max-w-2xl mx-auto px-6 py-2">
                    <div className="flex items-center gap-3">
                        <NavLink to={`/manga/${id}`} className="btn btn-sm shrink-0">
                            <ArrowLeft size={16} strokeWidth={3}/>
                            To Chapters
                        </NavLink>
                        <div className="flex-1 flex flex-col gap-1">
                            <p className={`text-xs text-center ${currentPage !== totalPages - 1 ? "text-neutral-content" : ""}`}>
                                {currentPage + 1} / <strong>{totalPages}</strong>
                            </p>
                            <progress className="progress progress-primary w-full" value={progressPct} max="100"/>
                        </div>
                        <div className="flex gap-2 shrink-0">
                            <button onClick={handlePageDown} disabled={currentPage === 0 && !prevChapter} className="btn btn-sm">
                                <ChevronLeft size={16} strokeWidth={4}/>
                            </button>
                            <button onClick={handlePageUp} disabled={currentPage >= totalPages - 1 && !nextChapter} className="btn btn-sm">
                                <ChevronRight size={16} strokeWidth={4}/>
                            </button>
                        </div>
                    </div>
                </div>
            </div>
            <div className="flex flex-col items-center">
                <div className="flex-1 overflow-hidden relative">
                    <img src={imageSrc} className="max-h-[calc(100vh-5rem)] max-w-full object-contain mt-2"/>
                    <div className="absolute inset-y-0 left-0 w-1/2 cursor-pointer hover:bg-black/10" onClick={handlePageDown}></div>
                    <div className="absolute inset-y-0 right-0 w-1/2 cursor-pointer hover:bg-black/10" onClick={handlePageUp}></div>
                </div>
            </div>
        </div>
    )
}