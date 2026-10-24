import { useEffect, useRef, useState } from "react"
import { Play, Pause, X, Music, SkipBack, SkipForward, Shuffle, Repeat, Repeat1, Volume2, VolumeX } from "lucide-react"
import useAudioStore, { REPEAT_LABELS } from "../../stores/useAudioStore"
import useSidebarStore from "../../stores/useSidebarStore"
import api from "../../services/api"
import { Link, useLocation } from "react-router-dom"
import { readStored, writeStored } from "../../utils/storage"
import { keyForFocused } from "../../utils/keys"
import { trackListening } from "../../utils/playCount"

// how far into a track Previous restarts it instead of going back a track
const RESTART_AFTER_SECS = 3
// how far the lock screen and headphone skip buttons jump
const SKIP_SECS = 10
// how often the position is remembered while playing, for picking up after a reload
const SAVE_EVERY_SECS = 5

// the cover and title, a link to the page the music is playing from when there is one
function TrackInfo({ source, title, children }) {
    const className = "flex items-center gap-3 min-w-0 flex-1 sm:flex-none sm:w-64"
    if (!source) return <div className={className}>{children}</div>
    return <Link to={source} aria-label={`${title}, go to where it's playing from`}
        className={`${className} -m-1.5 p-1.5 rounded-field hover:bg-white/5 active:bg-white/10 transition-colors`}>
        {children}
    </Link>
}

export default function AudioPlayer() {
    const { currentTrack, isPlaying, pause, resume, stop, queue, shuffle, repeat, toggleShuffle, cycleRepeat, source } = useAudioStore()
    // the bar starts where the sidebar ends
    const sidebarCollapsed = useSidebarStore(state => state.collapsed)
    // the video player, readers and login page fill the screen: no tab bar or sidebar to sit beside
    const { pathname } = useLocation()
    const onLogin = pathname === "/login"
    const inLayout = !pathname.startsWith("/watch/") && !pathname.endsWith("/read") && !onLogin
    const audioRef = useRef(null)
    const barRef = useRef(null)
    const visible = currentTrack !== null

    // --player-h is the bar's height (0px when hidden), pages pad their bottoms by it
    useEffect(() => {
        const root = document.documentElement
        const bar = barRef.current
        if (!bar) {
            root.style.setProperty("--player-h", "0px")
            return
        }
        const update = () => root.style.setProperty("--player-h", `${bar.offsetHeight}px`)
        update()
        const observer = new ResizeObserver(update)
        observer.observe(bar)
        return () => {
            observer.disconnect()
            root.style.setProperty("--player-h", "0px")
        }
    }, [visible])
    const [progress, setProgress] = useState(0)
    const [duration, setDuration] = useState(0)
    const [volume, setVolume] = useState(() => {
        const stored = Number(readStored("mediahub.volume", "1"))
        return Number.isFinite(stored) ? Math.min(Math.max(stored, 0), 1) : 1
    })
    const [muted, setMuted] = useState(() => readStored("mediahub.muted", "false") === "true")
    // a cover that failed to load, its signed link may have expired since the queue was saved
    const [failedThumb, setFailedThumb] = useState(null)

    const remintedRef = useRef(false)
    const lastSavedRef = useRef(0)
    // how long the current track has been listened to, for counting plays
    const listeningRef = useRef(null)

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

    // move through the queue. Landing on the same track (a queue of one) loads nothing new, so
    // it starts that track over instead
    function move(step) {
        const before = useAudioStore.getState().currentTrack?.id
        step()
        const audio = audioRef.current
        if (audio && useAudioStore.getState().currentTrack?.id === before) {
            audio.currentTime = 0
            if (useAudioStore.getState().isPlaying) audio.play().catch(() => {})
        }
    }

    function handleNext() {
        move(useAudioStore.getState().next)
    }

    // a few seconds in, Previous starts the track over, like other players
    function handlePrev() {
        const audio = audioRef.current
        if (audio && audio.currentTime > RESTART_AFTER_SECS) {
            audio.currentTime = 0
            return
        }
        move(useAudioStore.getState().prev)
    }

    function handleEnded() {
        const audio = audioRef.current
        if (useAudioStore.getState().repeat === "one" && audio) {
            audio.currentTime = 0
            audio.play().catch(() => {})
            return
        }
        handleNext()
    }

    function togglePlay() {
        if (useAudioStore.getState().isPlaying) pause()
        else resume()
    }

    // play/pause when store state changes, remembering where a pause left off
    useEffect(() => {
        const audio = audioRef.current
        if (!audio) return
        if (isPlaying) {
            audio.play().catch(() => {})
        } else {
            audio.pause()
            // before a track has loaded there is no position yet, saving 0 would lose a restored one
            if (audio.currentSrc) useAudioStore.getState().save(audio.currentTime)
        }
    }, [isPlaying])

    // load new track when currentTrack changes, from where a restored queue left off. Signed out,
    // the stream can't be fetched, so a track waits for the login page to be left
    const loadedRef = useRef(null)
    useEffect(() => {
        const audio = audioRef.current
        if (!audio || !currentTrack || onLogin) return
        if (loadedRef.current === currentTrack.id && audio.currentSrc) return
        loadedRef.current = currentTrack.id
        remintedRef.current = false
        const { resumeAt, save } = useAudioStore.getState()
        lastSavedRef.current = resumeAt
        save(resumeAt)
        loadTrack(currentTrack.id, resumeAt)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- only a different track loads, not a new copy of the same one
    }, [currentTrack?.id, onLogin])

    // signed out (or the session ran out): the bar hides and the music stops, the queue waits
    // for signing back in
    useEffect(() => {
        if (onLogin) pause()
    }, [onLogin, pause])

    // the saved volume, applied again whenever the bar's audio element is mounted
    useEffect(() => {
        if (!audioRef.current) return
        audioRef.current.volume = volume
        audioRef.current.muted = muted
    }, [volume, muted, visible])

    // the position is saved every few seconds while playing; this catches closing the tab between
    useEffect(() => {
        function saveNow() {
            if (audioRef.current) useAudioStore.getState().save(audioRef.current.currentTime)
        }
        window.addEventListener("pagehide", saveNow)
        return () => window.removeEventListener("pagehide", saveNow)
    }, [])

    // Space plays and pauses on library pages; the video player and readers have their own keys
    useEffect(() => {
        if (!visible || !inLayout) return
        function onKeyDown(e) {
            if (e.key !== " " || e.repeat || keyForFocused(e)) return
            e.preventDefault()
            togglePlay()
        }
        document.addEventListener("keydown", onKeyDown)
        return () => document.removeEventListener("keydown", onKeyDown)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- togglePlay reads the store when the key is pressed
    }, [visible, inLayout])

    // the lock screen, notification shade and headphone buttons control the bar too
    useEffect(() => {
        if (!("mediaSession" in navigator) || !visible) return
        const session = navigator.mediaSession
        const seekBy = secs => {
            const audio = audioRef.current
            if (audio) audio.currentTime = Math.min(Math.max(audio.currentTime + secs, 0), audio.duration || Infinity)
        }
        const handlers = {
            play: resume,
            pause,
            previoustrack: handlePrev,
            nexttrack: handleNext,
            seekbackward: e => seekBy(-(e.seekOffset || SKIP_SECS)),
            seekforward: e => seekBy(e.seekOffset || SKIP_SECS),
            seekto: e => { if (audioRef.current && e.seekTime != null) audioRef.current.currentTime = e.seekTime },
        }
        for (const [action, handler] of Object.entries(handlers)) {
            try {
                session.setActionHandler(action, handler)
            } catch {
                // a browser without this action just doesn't show its button
            }
        }
        return () => {
            for (const action of Object.keys(handlers)) {
                try { session.setActionHandler(action, null) } catch { /* not supported */ }
            }
        }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- the handlers read the store and audio element when they run
    }, [visible])

    useEffect(() => {
        if (!("mediaSession" in navigator)) return
        navigator.mediaSession.metadata = currentTrack ? new window.MediaMetadata({
            title: currentTrack.title,
            artist: currentTrack.artist ?? "",
            artwork: currentTrack.thumbnail ? [{ src: new URL(currentTrack.thumbnail, window.location.href).href }] : [],
        }) : null
    }, [currentTrack])

    useEffect(() => {
        if ("mediaSession" in navigator) navigator.mediaSession.playbackState = !visible ? "none" : isPlaying ? "playing" : "paused"
    }, [isPlaying, visible])

    function handleTimeUpdate() {
        const audio = audioRef.current
        if (!audio) return
        setProgress(audio.currentTime)
        setDuration(audio.duration || 0)

        const trackId = useAudioStore.getState().currentTrack?.id
        if (trackId) {
            const listening = trackListening(listeningRef.current, trackId, audio.currentTime, audio.duration || 0)
            listeningRef.current = listening.state
            if (listening.counted) api.post(`/music/${trackId}/plays`).catch(() => {})
        }

        if (Math.abs(audio.currentTime - lastSavedRef.current) >= SAVE_EVERY_SECS) {
            lastSavedRef.current = audio.currentTime
            useAudioStore.getState().save(audio.currentTime)
        }
        if ("mediaSession" in navigator && Number.isFinite(audio.duration) && audio.duration > 0) {
            try {
                navigator.mediaSession.setPositionState({
                    duration: audio.duration,
                    position: Math.min(audio.currentTime, audio.duration),
                    playbackRate: audio.playbackRate,
                })
            } catch {
                // the lock screen just doesn't show a position
            }
        }
    }

    function handleSeek(e) {
        if (!audioRef.current || !duration) return
        const rect = e.currentTarget.getBoundingClientRect()
        const pct = (e.clientX - rect.left) / rect.width
        audioRef.current.currentTime = pct * duration
    }

    function changeVolume(value) {
        setVolume(value)
        writeStored("mediahub.volume", String(value))
        if (muted && value > 0) toggleMute()
    }

    function toggleMute() {
        setMuted(!muted)
        writeStored("mediahub.muted", String(!muted))
    }

    function formatTime(secs) {
        const m = Math.floor(secs / 60)
        const s = Math.floor(secs % 60)
        return `${m}:${String(s).padStart(2, "0")}`
    }

    if (!currentTrack) return null

    const pct = duration ? (progress / duration) * 100 : 0
    const hasQueue = queue.length > 1
    const RepeatIcon = repeat === "one" ? Repeat1 : Repeat
    const repeatLabel = REPEAT_LABELS[repeat]

    return <>
        <audio
            ref={audioRef}
            onTimeUpdate={handleTimeUpdate}
            onEnded={handleEnded}
            onError={handleError}
            onPlaying={() => { remintedRef.current = false }}
        />
        {/* sits above the mobile tab bar, and beside the sidebar on desktop */}
        <div ref={barRef} className={`fixed right-0 left-0 z-50 transition-[left] duration-200 ${onLogin ? "hidden" : ""}
            ${inLayout ? "bottom-[calc(4rem+env(safe-area-inset-bottom))] lg:bottom-0" : "bottom-0 pb-[env(safe-area-inset-bottom)]"}
            ${!inLayout ? "" : sidebarCollapsed ? "lg:left-[4.5rem]" : "lg:left-60"}
            bg-base-300/90 backdrop-blur-xl border-t border-white/5 shadow-[0_-12px_32px_-12px_rgb(0_0_0/0.6)]`}>
            {/* phones: a thin progress line along the top edge, tap to seek */}
            <div className="sm:hidden absolute inset-x-0 top-0 h-4 -translate-y-1/2 cursor-pointer" onClick={handleSeek}>
                <div className="h-0.5 mt-[7px] bg-white/10">
                    <div className="h-full bg-primary" style={{ width: `${pct}%` }}/>
                </div>
            </div>

            <div className="flex items-center gap-3 sm:gap-4 px-3 sm:px-4 py-2">
                {/* Track Info: opens the album or playlist it's playing from */}
                <TrackInfo source={source} title={currentTrack.title}>
                    {currentTrack.thumbnail && failedThumb !== currentTrack.thumbnail ? (
                        <img src={currentTrack.thumbnail} alt="" onError={() => setFailedThumb(currentTrack.thumbnail)}
                            className="size-11 shrink-0 rounded-lg object-cover shadow-card" />
                    ) : (
                        <div className="size-11 rounded-lg bg-base-200 flex items-center justify-center shrink-0">
                            <Music size={18} />
                        </div>
                    )}
                    <div className="min-w-0">
                        <p className="text-sm font-medium truncate">{currentTrack.title}</p>
                        <p className="text-xs text-base-content/60 truncate">{currentTrack.artist}</p>
                    </div>
                </TrackInfo>

                {/* Controls */}
                <div className="sm:flex-1 flex flex-col items-center gap-1">
                    <div className="flex items-center gap-1 sm:gap-3">
                        {hasQueue && (
                            <button className={`hidden sm:inline-flex btn btn-ghost btn-sm btn-circle ${shuffle ? "text-primary" : "text-base-content/60"}`}
                                onClick={toggleShuffle} aria-label={shuffle ? "Turn shuffle off" : "Shuffle"} aria-pressed={shuffle}>
                                <Shuffle size={16} />
                            </button>
                        )}
                        {hasQueue && (
                            <button className="btn btn-ghost btn-sm btn-circle" onClick={handlePrev} aria-label="Previous track">
                                <SkipBack size={16} />
                            </button>
                        )}
                        <button className="btn btn-circle btn-primary shadow-glow"
                            onClick={togglePlay}
                            aria-label={isPlaying ? "Pause" : "Play"}>
                            {isPlaying ? <Pause size={18} fill="currentColor" /> : <Play size={18} fill="currentColor" className="ml-0.5" />}
                        </button>
                        {hasQueue && (
                            <button className="btn btn-ghost btn-sm btn-circle" onClick={handleNext} aria-label="Next track">
                                <SkipForward size={16} />
                            </button>
                        )}
                        <button className={`hidden sm:inline-flex btn btn-ghost btn-sm btn-circle ${repeat !== "off" ? "text-primary" : "text-base-content/60"}`}
                            onClick={cycleRepeat} aria-label={repeatLabel} aria-pressed={repeat !== "off"}>
                            <RepeatIcon size={16} />
                        </button>
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

                {/* Volume: phones use their own buttons */}
                <div className="hidden md:flex items-center gap-1 w-36">
                    <button className="btn btn-ghost btn-sm btn-circle text-base-content/60" onClick={toggleMute} aria-label={muted ? "Unmute" : "Mute"}>
                        {muted || volume === 0 ? <VolumeX size={16} /> : <Volume2 size={16} />}
                    </button>
                    <input type="range" min={0} max={1} step={0.01} value={muted ? 0 : volume}
                        onChange={e => changeVolume(Number(e.target.value))}
                        className="range range-primary range-xs flex-1" aria-label="Volume" />
                </div>

                {/* Close */}
                <button className="btn btn-ghost btn-sm btn-circle" onClick={stop} aria-label="Close player">
                    <X size={16} />
                </button>
            </div>
        </div>
    </>
}
