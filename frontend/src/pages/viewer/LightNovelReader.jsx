import { ChevronDown, ChevronUp, ArrowLeft } from 'lucide-react'
import { useState, useEffect, useRef } from "react"
import { useParams, NavLink } from "react-router-dom"
import api from "../../services/api"
import { useMediaItem } from "../../hooks"
import Loading from "../../components/states/Loading"
import Error from "../../components/states/Error"

export default function LightNovelReader() {
    const { id, volumeId } = useParams()
    const { item: ln, loading, error } = useMediaItem(id)

    if (loading) return <Loading />
    if (error) return <Error error={error} />
    if (!ln) return null

    const volume = ln.metadata?.volumes?.find(v => v.id === volumeId)
    return <LightNovelReaderInner id={id} volumeId={volumeId} volume={volume} />
}

function LightNovelReaderInner({ id, volumeId, volume }) {
    const initialScroll = volume?.scroll_position ?? 0;

    const [content, setContent] = useState(null)
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState(null)
    const [scrollPct, setScrollPct] = useState(initialScroll)
    const contentRef = useRef(null)
    const scrollPctRef = useRef(initialScroll)
    const debounceTimer = useRef(null)
    const hasRestored = useRef(false)

    useEffect(() => {
        setLoading(true)
        api.get(`/light-novels/${id}/volumes/${volumeId}/content`)
            .then(res => setContent(res.data))
            .catch(err => setError(err.message))
            .finally(() => setLoading(false))
    }, [volumeId])

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
            api.put(`/light-novels/volumes/${volumeId}/read`, { read: true })
        }
    }

    if (loading) return <Loading />
    if (error) return <Error error={error} />

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
                        <NavLink to={`/light-novels/${id}`} className="btn btn-sm">
                            <ArrowLeft size={16} strokeWidth={3}/>
                            To Volumes
                        </NavLink>
                        <button
                            className="btn btn-sm"
                            disabled={scrollPct <= 0}
                            onClick={() => contentRef.current?.scrollTo({ top: 0, behavior: 'smooth' })}>
                                <ChevronUp size={20} strokeWidth={3}/>
                                Top
                        </button>
                        <button
                            className="btn btn-sm"
                            disabled={scrollPct >= 1}
                            onClick={() => {
                                const el = contentRef.current;
                                if (el) el.scrollTo({ top: el.scrollHeight, behavior: 'smooth' });
                            }}>
                                <ChevronDown size={20} strokeWidth={3}/>
                                Bottom
                        </button>
                    </div>
                </div>
            </div>
            <div
                ref={contentRef}
                className="flex-1 overflow-y-auto flex flex-col items-center"
            >
                <div className="w-full max-w-2xl px-6 py-4">
                    <div
                        className="prose prose-invert max-w-none bg-white text-black text-xs px-2"
                        dangerouslySetInnerHTML={{ __html: content }}
                    />
                </div>
            </div>
        </div>
    )
}