import { useEffect, useRef, useState } from "react"
import { Play, Pause, X, Music, SkipBack, SkipForward } from "lucide-react"
import useAudioStore from "../../stores/useAudioStore"

export default function AudioPlayer() {
    const { currentTrack, isPlaying, pause, resume, stop, next, prev, queue } = useAudioStore()
    const audioRef = useRef(null)
    const [progress, setProgress] = useState(0)
    const [duration, setDuration] = useState(0)

    const baseURL = import.meta.env.VITE_API_URL

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
        audioRef.current.src = `${baseURL}/stream/music/${currentTrack.id}`
        audioRef.current.play().catch(() => {})
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

    return <>
        <audio ref={audioRef} onTimeUpdate={handleTimeUpdate} onEnded={next} />
        <div className="fixed bottom-0 left-0 right-0 bg-base-300 border-t border-base-content/10 px-4 py-2 flex items-center gap-4 z-50">
            {/* Track Info */}
            <div className="flex items-center gap-3 min-w-0 w-64">
                {currentTrack.thumbnail ? (
                    <img src={currentTrack.thumbnail} className="w-10 h-10 rounded object-cover" />
                ) : (
                    <div className="w-10 h-10 rounded bg-base-200 flex items-center justify-center">
                        <Music size={18} />
                    </div>
                )}
                <div className="min-w-0">
                    <p className="text-sm font-medium truncate">{currentTrack.title}</p>
                    <p className="text-xs opacity-70 truncate">{currentTrack.artist}</p>
                </div>
            </div>

            {/* Controls */}
            <div className="flex-1 flex flex-col items-center gap-1">
                <div className="flex items-center gap-3">
                    {queue.length > 0 && (
                        <button className="btn btn-ghost btn-sm" onClick={prev}>
                            <SkipBack size={14} />
                        </button>
                    )}
                    <button className="btn btn-circle btn-sm btn-primary"
                        onClick={() => isPlaying ? pause() : resume()}>
                        {isPlaying ? <Pause size={16} /> : <Play size={16} />}
                    </button>
                    {queue.length > 0 && (
                        <button className="btn btn-ghost btn-sm" onClick={next}>
                            <SkipForward size={14} />
                        </button>
                    )}
                </div>

                {/* Progress Bar */}
                <div className="flex items-center gap-2 w-full max-w-lg">
                    <span className="text-xs opacity-50 w-10 text-right">{formatTime(progress)}</span>
                    <div
                        className="flex-1 h-1.5 bg-base-content/20 rounded-full cursor-pointer"
                        onClick={handleSeek}
                    >
                        <div
                            className="h-full bg-primary rounded-full"
                            style={{ width: `${duration ? (progress / duration) * 100 : 0}%` }}
                        />
                    </div>
                    <span className="text-xs opacity-50 w-10">{formatTime(duration)}</span>
                </div>
            </div>

            {/* Close */}
            <button className="btn btn-ghost btn-sm" onClick={stop}>
                <X size={16} />
            </button>
        </div>
    </>
}