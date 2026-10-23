import { useRef, useEffect, useCallback } from "react";
import { useLocation, useNavigate, useParams } from "react-router-dom"
import { ChevronLeft, ChevronRight } from "lucide-react"
import { useMediaItem, usePages } from "../../hooks";
import ErrorState from "../../components/states/ErrorState";
import NotFound from "../../components/states/NotFound";
import { PageSkeleton, ReaderSkeleton } from "../../components/states/Skeletons";
import api from "../../services/api";
import { exitViewer, switchViewer } from "../../utils/navigation";
import ReaderToolbar from "../../components/ReaderToolbar";
import useDocumentTitle from "../../hooks/useDocumentTitle";

export default function MangaReader() {
    const { id, chapterId } = useParams();
    const { item, loading, error, notFound, refetch } = useMediaItem(id);

    if (loading) return <ReaderSkeleton variant="page" />
    if (notFound) return <NotFound title="Manga not found" hint="This manga isn't in the library any more." />
    if (error) return <ErrorState error={error} title="Couldn't open this manga" onRetry={refetch} showBack />
    if (!item) return null

    const chapters = item.metadata?.chapters ?? [];
    const chapter = chapters.find(c => c.id === chapterId);
    if (!chapter) return <NotFound title="Chapter not found" hint="This chapter isn't part of the manga any more. Pick another from the chapter list." />
    return <MangaReaderInner key={chapterId} id={id} chapterId={chapterId} chapter={chapter} chapters={chapters} title={item.title} />
}

function MangaReaderInner({ id, chapterId, chapter, chapters, title }) {
    useDocumentTitle(`${title} · Ch. ${chapter.chapter_number}`);
    const navigate = useNavigate();
    const location = useLocation();
    const initialPage = chapter?.last_page_read ?? 0;
    const totalPages = chapter?.page_count ?? 0;
    const { currentPage, setCurrentPage, imageSrc, loading, error, refetch } = usePages(id, chapterId, initialPage, totalPages);
    const pagesSinceLastSave = useRef(0);
    const currentPageRef = useRef(currentPage);
    const currentIndex = chapters.findIndex(c => c.id === chapterId);
    const nextChapter = chapters[currentIndex + 1] ?? null;
    const prevChapter = chapters[currentIndex - 1] ?? null;

    useEffect(() => {
        currentPageRef.current = currentPage;
    }, [currentPage]);

    const saveProgress = useCallback(page => {
        const completed = page >= totalPages - 1;
        api.put(`/manga/${id}/chapters/${chapterId}/progress`, {
            last_page_read: page,
            completed,
        }).catch(() => {});
        pagesSinceLastSave.current = 0;
        if (completed) {
            // best effort like the progress save above, reading carries on either way
            api.put(`/manga/chapters/${chapterId}/read`, { read: true }).catch(() => {});
        }
    }, [id, chapterId, totalPages]);

    // saves once on leaving the chapter, with the page count known by then
    const saveProgressRef = useRef(saveProgress);
    useEffect(() => {
        saveProgressRef.current = saveProgress;
    }, [saveProgress]);
    useEffect(() => {
        return () => saveProgressRef.current(currentPageRef.current);
    }, []);

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
            switchViewer(navigate, location, `/manga/${id}/chapters/${nextChapter.id}/read`);
        }
    }

    function handlePageDown() {
        if (currentPage - 1 >= 0) {
            changePage(currentPage - 1);
        } else if (prevChapter) {
            saveProgress(currentPage);
            switchViewer(navigate, location, `/manga/${id}/chapters/${prevChapter.id}/read`);
        }
    }

    // bound again on every render, so the keys always turn from the page on screen
    useEffect(() => {
        function handleKeyDown(e) {
            if (e.key === "ArrowRight") handlePageUp();
            if (e.key === "ArrowLeft") handlePageDown();
        }
        window.addEventListener("keydown", handleKeyDown);
        return () => window.removeEventListener("keydown", handleKeyDown);
    });

    const progressPct = totalPages ? Math.round(((currentPage + 1) / totalPages) * 100) : 0;

    return (
        <div className="flex flex-col min-h-dvh bg-black/30 pb-[var(--player-h,0px)]">
            <ReaderToolbar
                onBack={() => exitViewer(navigate, location, `/manga/${id}`)}
                backLabel="Chapters"
                title={title}
                subtitle={chapter.title ?? `Chapter ${chapter.chapter_number}`}
                progress={progressPct / 100}
                progressLabel={totalPages ? `Page ${currentPage + 1} of ${totalPages}` : null}
            >
                <button onClick={handlePageDown} disabled={currentPage === 0 && !prevChapter} className="btn btn-ghost btn-sm btn-square"
                    aria-label={currentPage === 0 ? "Previous chapter" : "Previous page"}>
                    <ChevronLeft size={20}/>
                </button>
                <button onClick={handlePageUp} disabled={currentPage >= totalPages - 1 && !nextChapter} className="btn btn-ghost btn-sm btn-square"
                    aria-label={currentPage >= totalPages - 1 ? "Next chapter" : "Next page"}>
                    <ChevronRight size={20}/>
                </button>
            </ReaderToolbar>
            <div className="flex-1 flex flex-col items-center">
                {/* the toolbar stays while a page loads or fails, so the reader can still move on */}
                {error ? (
                    <div className="max-w-xl w-full mt-8 px-4">
                        <ErrorState compact error={error} onRetry={refetch}/>
                    </div>
                ) : loading ? (
                    <PageSkeleton />
                ) : <div className="relative select-none">
                    <img src={imageSrc} alt={`Page ${currentPage + 1}`} className="block max-h-[calc(100dvh-3.75rem-env(safe-area-inset-top)-var(--player-h,0px))] max-w-full object-contain shadow-raised"/>
                    {/* tap the left or right half to turn the page */}
                    <div className="absolute inset-y-0 left-0 w-1/2 cursor-w-resize" onClick={handlePageDown}></div>
                    <div className="absolute inset-y-0 right-0 w-1/2 cursor-e-resize" onClick={handlePageUp}></div>
                </div>}
            </div>
        </div>
    )
}