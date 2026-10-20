import { ChevronDown, ChevronUp } from 'lucide-react'
import { useState, useEffect, useMemo, useRef } from "react"
import { useParams, useLocation, useNavigate } from "react-router-dom"
import api from "../../services/api"
import { useMediaItem } from "../../hooks"
import useApi from "../../hooks/useApi"
import ErrorState from "../../components/states/ErrorState"
import NotFound from "../../components/states/NotFound"
import { ReaderSkeleton, TextSkeleton } from "../../components/states/Skeletons"
import { exitViewer, switchViewer } from "../../utils/navigation"
import ReaderToolbar from "../../components/ReaderToolbar"
import { sanitizeHtml } from "../../utils/sanitize"

export default function LightNovelReader() {
    const { id, volumeId } = useParams()
    const { item: ln, loading, error, notFound, refetch } = useMediaItem(id)

    if (loading) return <ReaderSkeleton variant="text" />
    if (notFound) return <NotFound title="Light novel not found" hint="This light novel isn't in the library any more." />
    if (error) return <ErrorState error={error} title="Couldn't open this light novel" onRetry={refetch} showBack />
    if (!ln) return null

    const volumes = ln.metadata?.volumes ?? [];
    const volume = volumes.find(v => v.id === volumeId);
    if (!volume) return <NotFound title="Volume not found" hint="This volume isn't part of the light novel any more. Pick another from the volume list." />
    return <LightNovelReaderInner key={volumeId} id={id} volumeId={volumeId} volume={volume} volumes={volumes} title={ln.title} />
}

function LightNovelReaderInner({ id, volumeId, volume, volumes, title }) {
    const navigate = useNavigate();
    const location = useLocation();
    const initialScroll = volume?.scroll_position ?? 0;

    const { data: content, loading, error, reload } = useApi(`/light-novels/${id}/volumes/${volumeId}/content`);
    // an EPUB is a file from anywhere: its markup can't be allowed to run code in the page
    const safeContent = useMemo(() => sanitizeHtml(content), [content]);
    const [scrollPct, setScrollPct] = useState(initialScroll);
    const contentRef = useRef(null);
    const scrollPctRef = useRef(initialScroll);
    const debounceTimer = useRef(null);
    const hasRestored = useRef(false);
    const currentIndex = volumes.findIndex(v => v.id === volumeId);
    const nextVolume = volumes[currentIndex + 1] ?? null;
    const prevVolume = volumes[currentIndex - 1] ?? null;


    // restore scroll position after content renders
    useEffect(() => {
        if (!content || hasRestored.current || initialScroll === 0) return
        const el = contentRef.current
        if (!el) return
        setTimeout(() => {
            const maxScroll = el.scrollHeight - el.clientHeight;
            el.scrollTop = initialScroll * maxScroll;
            hasRestored.current = true;
        }, 150)
    }, [content])

    // track scroll position
    useEffect(() => {
        const el = contentRef.current
        if (!el) return

        function handleScroll() {
            const maxScroll = el.scrollHeight - el.clientHeight
            const pct = maxScroll > 0 ? el.scrollTop / maxScroll : 0
            scrollPctRef.current = pct
            setScrollPct(pct)

            clearTimeout(debounceTimer.current)
            debounceTimer.current = setTimeout(() => {
                saveProgress(pct)
            }, 2000)
        }

        el.addEventListener("scroll", handleScroll)
        return () => {
            el.removeEventListener("scroll", handleScroll)
            clearTimeout(debounceTimer.current)
        }
    }, [content])

    // save on unmount
    useEffect(() => {
        return () => {
            saveProgress(scrollPctRef.current)
        }
    }, [volumeId])

    function saveProgress(pct) {
        api.put(`/light-novels/volumes/${volumeId}/progress`, {
            scroll_position: pct,
        }).catch(() => {})
        if (pct >= 1) {
            // best effort like the progress save above, reading carries on either way
            api.put(`/light-novels/volumes/${volumeId}/read`, { read: true }).catch(() => {})
        }
    }

    if (error) return <ErrorState error={error} title="Couldn't load this volume" onRetry={reload} showBack />

    const atStart = scrollPct <= 0
    const atEnd = scrollPct >= 1

    return (
        <div className="flex flex-col h-dvh">
            <ReaderToolbar
                onBack={() => exitViewer(navigate, location, `/light-novels/${id}`)}
                backLabel="Volumes"
                title={title}
                subtitle={volume.title ?? `Volume ${volume.volume_number}`}
                progress={scrollPct}
                progressLabel={`${Math.round(scrollPct * 100)}%`}
            >
                <button
                    className="btn btn-ghost btn-sm gap-1"
                    disabled={atStart && !prevVolume}
                    aria-label={atStart && prevVolume ? "Previous volume" : "Back to top"}
                    onClick={() => {
                        if (atStart && prevVolume) {
                            saveProgress(scrollPctRef.current);
                            switchViewer(navigate, location, `/light-novels/${id}/volumes/${prevVolume.id}/read`);
                        } else {
                            contentRef.current?.scrollTo({ top: 0, behavior: 'smooth' });
                        }
                    }}>
                    <ChevronUp size={18}/>
                    <span className={atStart && prevVolume ? "" : "hidden md:inline"}>{atStart && prevVolume ? "Prev volume" : "Top"}</span>
                </button>
                <button
                    className={`btn btn-sm gap-1 ${atEnd && nextVolume ? "btn-primary" : "btn-ghost"}`}
                    disabled={atEnd && !nextVolume}
                    aria-label={atEnd && nextVolume ? "Next volume" : "Jump to the end"}
                    onClick={() => {
                        if (atEnd && nextVolume) {
                            saveProgress(scrollPctRef.current);
                            switchViewer(navigate, location, `/light-novels/${id}/volumes/${nextVolume.id}/read`);
                        } else {
                            const el = contentRef.current;
                            if (el) el.scrollTo({ top: el.scrollHeight, behavior: 'smooth' });
                        }
                    }}>
                    <ChevronDown size={18}/>
                    <span className={atEnd && nextVolume ? "" : "hidden md:inline"}>{atEnd && nextVolume ? "Next volume" : "End"}</span>
                </button>
            </ReaderToolbar>
            {/* the toolbar stays up while the text loads */}
            {loading ? <TextSkeleton /> : <div
                ref={contentRef}
                className="flex-1 overflow-y-auto flex flex-col items-center px-3 sm:px-6 py-4 sm:py-8 pb-[calc(2rem+var(--player-h,0px)+env(safe-area-inset-bottom))]"
            >
                {/* a warm page to read on, the text comes with its own markup from the EPUB */}
                <article
                    className="w-full max-w-2xl rounded-box bg-[#f7f2e8] text-[#1f1a14] shadow-raised px-5 sm:px-12 py-8 sm:py-12
                        text-[1.0625rem] leading-relaxed font-[Georgia,'Times_New_Roman',serif]
                        [&_p]:mb-4 [&_img]:max-w-full [&_img]:h-auto [&_img]:mx-auto [&_img]:my-6 [&_img]:rounded
                        [&_h1]:text-2xl [&_h1]:font-semibold [&_h1]:mb-4 [&_h2]:text-xl [&_h2]:font-semibold [&_h2]:mb-3 [&_h3]:font-semibold [&_h3]:mb-2"
                    dangerouslySetInnerHTML={{ __html: safeContent }}
                />
            </div>}
        </div>
    )
}