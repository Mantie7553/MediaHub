import { useEffect, useRef, useState } from "react"
import { Play, Pause, X, Music, SkipBack, SkipForward } from "lucide-react"
import useAudioStore from "../../stores/useAudioStore"
import useSidebarStore from "../../stores/useSidebarStore"
import api from "../../services/api"
import { useLocation } from "react-router-dom"

export default function AudioPlayer() {
    const { currentTrack, isPlaying, pause, resume, stop, next, prev, queue } = useAudioStore()
    // the bar starts where the sidebar ends
    const sidebarCollapsed = useSidebarStore(state => state.collapsed)
    // the video player and login page have no tab bar to sit above
    const { pathname } = useLocation()
    const hasTabBar = !pathname.startsWith("/watch/") && pathname !== "/login"
    const audioRef = useRef(null)
    const [progress, setProgress] = useState(0)
    const [duration, setDuration] = useState(0)

    const remintedRef = useRef(false)

    const baseURL = import.meta.env.VITE_API_URL

    // the audio element can't send the Bearer header, so fetch a signed URL for the track
    async function loadTrack(trackId, resumeAt = 0) {
        let res
        try {
            res = await api.get(`/stream/music/${trackId}/url`)
        } catch {
            return
        }

        const audio = audioRef.current
        // the track may have changed while the URL was being fetched
        if (!audio || useAudioStore.getState().currentTrack?.id !== trackId) return

        audio.src = `${baseURL}${res.data.url}`
        if (resumeAt > 0) {
            audio.addEventListener("loadedmetadata", () => {
                audio.currentTime = resumeAt
            }, { once: true })
        }
        if (useAudioStore.getState().isPlaying) {
            audio.play().catch(() => {})
        }
    }

    // signed URL expired (or the load failed), get a new one once and resume where we were
    function handleError() {
        if (!audioRef.current || !currentTrack || remintedRef.current) return
        remintedRef.current = true
        loadTrack(currentTrack.id, audioRef.current.currentTime)
    }

    // play/pause when store state changes
    useEffect(() => {
        if (!audioRef.current) return
        if (isPlaying) {
            audioRef.current.play().catch(() => {})
        } else {
            audioRef.current.pause()
        }
    }, [isPlaying])

    // load new track when currentTrack changes
    useEffect(() => {
        if (!audioRef.current || !currentTrack) return
        remintedRef.current = false
        loadTrack(currentTrack.id)
    }, [currentTrack?.id])

    function handleTimeUpdate() {
        if (!audioRef.current) return
        setProgress(audioRef.current.currentTime)
        setDuration(audioRef.current.duration || 0)
    }

    function handleSeek(e) {
        if (!audioRef.current || !duration) return
        const rect = e.currentTarget.getBoundingClientRect()
        const pct = (e.clientX - rect.left) / rect.width
        audioRef.current.currentTime = pct * duration
    }

    function formatTime(secs) {
        const m = Math.floor(secs / 60)
        const s = Math.floor(secs % 60)
        return `${m}:${String(s).padStart(2, "0")}`
    }

    if (!currentTrack) return null

    const pct = duration ? (progress / duration) * 100 : 0

    return <>
        <audio
            ref={audioRef}
            onTimeUpdate={handleTimeUpdate}
            onEnded={next}
            onError={handleError}
            onPlaying={() => { remintedRef.current = false }}
        />
        {/* sits above the mobile tab bar, and beside the sidebar on desktop */}
        <div className={`fixed right-0 left-0 z-50 transition-[left] duration-200
            ${hasTabBar ? "bottom-[calc(4rem+env(safe-area-inset-bottom))]" : "bottom-0 pb-[env(safe-area-inset-bottom)]"} lg:bottom-0 lg:pb-0
            ${sidebarCollapsed ? "lg:left-[4.5rem]" : "lg:left-60"}
            bg-base-300/90 backdrop-blur-xl border-t border-white/5 shadow-[0_-12px_32px_-12px_rgb(0_0_0/0.6)]`}>
            {/* phones: a thin progress line along the top edge, tap to seek */}
            <div className="sm:hidden absolute inset-x-0 top-0 h-4 -translate-y-1/2 cursor-pointer" onClick={handleSeek}>
                <div className="h-0.5 mt-[7px] bg-white/10">
                    <div className="h-full bg-primary" style={{ width: `${pct}%` }}/>
                </div>
            </div>

            <div className="flex items-center gap-3 sm:gap-4 px-3 sm:px-4 py-2">
                {/* Track Info */}
                <div className="flex items-center gap-3 min-w-0 flex-1 sm:flex-none sm:w-64">
                    {currentTrack.thumbnail ? (
                        <img src={currentTrack.thumbnail} className="size-11 rounded-lg object-cover shadow-card" />
                    ) : (
                        <div className="size-11 rounded-lg bg-base-200 flex items-center justify-center shrink-0">
                            <Music size={18} />
                        </div>
                    )}
                    <div className="min-w-0">
                        <p className="text-sm font-medium truncate">{currentTrack.title}</p>
                        <p className="text-xs text-base-content/60 truncate">{currentTrack.artist}</p>
                    </div>
                </div>

                {/* Controls */}
                <div className="sm:flex-1 flex flex-col items-center gap-1">
                    <div className="flex items-center gap-1 sm:gap-3">
                        {queue.length > 0 && (
                            <button className="hidden sm:inline-flex btn btn-ghost btn-sm btn-circle" onClick={prev} aria-label="Previous track">
                                <SkipBack size={16} />
                            </button>
                        )}
                        <button className="btn btn-circle btn-primary shadow-glow"
                            onClick={() => isPlaying ? pause() : resume()}
                            aria-label={isPlaying ? "Pause" : "Play"}>
                            {isPlaying ? <Pause size={18} fill="currentColor" /> : <Play size={18} fill="currentColor" className="ml-0.5" />}
                        </button>
                        {queue.length > 0 && (
                            <button className="btn btn-ghost btn-sm btn-circle" onClick={next} aria-label="Next track">
                                <SkipForward size={16} />
                            </button>
                        )}
                    </div>

                    {/* Progress Bar */}
                    <div className="hidden sm:flex items-center gap-2 w-full max-w-lg">
                        <span className="text-xs text-base-content/50 w-10 text-right tabular-nums">{formatTime(progress)}</span>
                        <div className="group/seek flex-1 py-1.5 cursor-pointer" onClick={handleSeek}>
                            <div className="h-1 group-hover/seek:h-1.5 transition-all bg-white/15 rounded-full">
                                <div className="h-full bg-primary rounded-full" style={{ width: `${pct}%` }} />
                            </div>
                        </div>
                        <span className="text-xs text-base-content/50 w-10 tabular-nums">{formatTime(duration)}</span>
                    </div>
                </div>

                {/* Close */}
                <button className="btn btn-ghost btn-sm btn-circle" onClick={stop} aria-label="Close player">
                    <X size={16} />
                </button>
            </div>
        </div>
    </>
}
