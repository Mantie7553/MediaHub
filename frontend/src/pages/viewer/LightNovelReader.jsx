import { ChevronDown, ChevronUp, ArrowLeft } from 'lucide-react'
import { useState, useEffect, useRef } from "react"
import { useParams, useLocation, useNavigate } from "react-router-dom"
import api from "../../services/api"
import { useMediaItem } from "../../hooks"
import useApi from "../../hooks/useApi"
import ErrorState from "../../components/states/ErrorState"
import NotFound from "../../components/states/NotFound"
import { ReaderSkeleton, TextSkeleton } from "../../components/states/Skeletons"
import { exitViewer, switchViewer } from "../../utils/navigation"

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
    return <LightNovelReaderInner key={volumeId} id={id} volumeId={volumeId} volume={volume} volumes={volumes} />
}

function LightNovelReaderInner({ id, volumeId, volume, volumes }) {
    const navigate = useNavigate();
    const location = useLocation();
    const initialScroll = volume?.scroll_position ?? 0;

    const { data: content, loading, error, reload } = useApi(`/light-novels/${id}/volumes/${volumeId}/content`);
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

    return (
        <div className="flex flex-col h-screen">
            <div className="sticky top-0 z-10 bg-base-200 border-b border-base-300">
                <div className="max-w-2xl mx-auto px-6">
                    <div className="flex justify-center pt-2">
                        <span className="text-xs text-neutral-content">{Math.round(scrollPct * 100)}%</span>
                    </div>
                    <div className="relative w-full h-1 my-1">
                        <progress
                            className="progress progress-primary w-full h-1"
                            value={Math.round(scrollPct * 100)}
                            max="100"
                        />
                    </div>
                    <div className="flex justify-center items-center gap-4 p-3">
                        <button onClick={() => exitViewer(navigate, location, `/light-novels/${id}`)} className="btn btn-sm">
                            <ArrowLeft size={16} strokeWidth={3}/>
                            To Volumes
                        </button>
                        <button
                            className="btn btn-sm"
                            disabled={scrollPct <= 0 && !prevVolume}
                            onClick={() => {
                                if (scrollPct <= 0 && prevVolume) {
                                    saveProgress(scrollPctRef.current);
                                    switchViewer(navigate, location, `/light-novels/${id}/volumes/${prevVolume.id}/read`);
                                } else {
                                    contentRef.current?.scrollTo({ top: 0, behavior: 'smooth' });
                                }
                            }}>
                            <ChevronUp size={20} strokeWidth={3}/>
                            {scrollPct <= 0 && prevVolume ? "Prev Volume" : "Top"}
                        </button>
                        <button
                            className="btn btn-sm"
                            disabled={scrollPct >= 1 && !nextVolume}
                            onClick={() => {
                                if (scrollPct >= 1 && nextVolume) {
                                    saveProgress(scrollPctRef.current);
                                    switchViewer(navigate, location, `/light-novels/${id}/volumes/${nextVolume.id}/read`);
                                } else {
                                    const el = contentRef.current;
                                    if (el) el.scrollTo({ top: el.scrollHeight, behavior: 'smooth' });
                                }
                            }}>
                            <ChevronDown size={20} strokeWidth={3}/>
                            {scrollPct >= 1 && nextVolume ? "Next Volume" : "Bottom"}
                        </button>
                    </div>
                </div>
            </div>
            {/* the toolbar stays up while the text loads */}
            {loading ? <TextSkeleton /> : <div
                ref={contentRef}
                className="flex-1 overflow-y-auto flex flex-col items-center"
            >
                <div className="w-full max-w-2xl px-6 py-4">
                    <div
                        className="prose prose-invert max-w-none bg-white text-black text-xs px-2"
                        dangerouslySetInnerHTML={{ __html: content }}
                    />
                </div>
            </div>}
        </div>
    )
}