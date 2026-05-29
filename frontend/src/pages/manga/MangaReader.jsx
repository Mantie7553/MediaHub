import { useEffect } from "react";
import { useParams } from "react-router-dom"
import { useMediaItem, usePages } from "../../hooks";
import Loading from "../../components/states/Loading";
import Error from "../../components/states/Error";

/**
 * This page displays a given mangas pages for reading
 * @returns 
 */
export default function MangaReader() {
    const { id, chapterId } = useParams();
    const { item, loading: mangaLoading, error: mangaError } = useMediaItem(id);
    const { currentPage, setCurrentPage, imageSrc, loading: pageLoading, error: pageError } = usePages(id, chapterId);
    const totalPages = item?.metadata?.chapters?.find(c => c.id === chapterId)?.page_count ?? 0;

    function handlePageUp() {
        setCurrentPage(prev => prev + 1 < totalPages ? prev + 1 : prev)
    }

    function handlePageDown() {
        setCurrentPage(prev => prev - 1 >= 0 ? prev - 1 : prev)
    }

    // allow for changing pages with key presses
    useEffect(() => {
        function handleKeyDown(e) {
            if (e.key === "ArrowRight") setCurrentPage(prev => prev + 1 < totalPages ? prev + 1 : prev);
            if (e.key === "ArrowLeft") setCurrentPage(prev => prev - 1 >= 0 ? prev - 1 : prev);
        }
        window.addEventListener("keydown", handleKeyDown);
        return () => window.removeEventListener("keydown", handleKeyDown);
    }, [totalPages]);

    if (mangaLoading || pageLoading) return <Loading />
    if (mangaError || pageError) return <Error error={mangaError || pageError} />
    
    return <div className="flex flex-col items-center">
        <div className="flex-1 overflow-hidden relative">
            <img src={imageSrc}/>
            <div className="absolute inset-y-0 left-0 w-1/2 cursor-pointer hover:bg-black/10" onClick={handlePageDown}></div>
            <div className="absolute inset-y-0 right-0 w-1/2 cursor-pointer hover:bg-black/10" onClick={handlePageUp}></div>
        </div>
        <div className="flex gap-2 items-center">
            <button onClick={handlePageDown} disabled={currentPage === 0} className="btn">Prev</button>
            <p className={currentPage !== totalPages ? "text-neutral-content" : ""}>{currentPage + 1} / <strong>{totalPages}</strong></p>
            <button onClick={handlePageUp} disabled={currentPage >= totalPages - 1} className="btn">Next</button>
        </div>
    </div>
}