import { useState, useEffect, useRef } from "react"
import { useParams, useLocation, useNavigate } from "react-router-dom"
import Hls from "hls.js"
import api from "../../services/api"
import ErrorState from "../../components/states/ErrorState"
import { toast } from "../../stores/useToastStore"
import { exitViewer, switchViewer } from "../../utils/navigation"
import { keyForFocused } from "../../utils/keys"
import useDocumentTitle from "../../hooks/useDocumentTitle"
import useAssRenderer from "../../hooks/useAssRenderer"
import { subtitleMode } from "../../utils/subtitles"
import { watchedBetween } from "../../utils/watchTime"
import { ArrowLeft, Languages, SkipBack, SkipForward, X } from "lucide-react"

// how long the next episode waits after one ends, and how far the arrow keys skip
const UP_NEXT_SECS = 5
const SKIP_SECS = 10

export default function PlayerPage() {
    const location = useLocation();
    const navigate = useNavigate();
    const { type, id } = useParams();
    const videoRef = useRef(null);
    const [playlistUrl, setPlaylistUrl] = useState(null);
    // [{ index, label, language, selected }]
    const [audioTracks, setAudioTracks] = useState([]);
    // [{ index, url, assUrl, burn, label, language }], url is null for image tracks that can
    // only be burned in, assUrl is set for ASS tracks that can be shown with their styling
    const [subtitles, setSubtitles] = useState([]);
    // subtitle url -> downloaded file (object URL) once it exists
    const [subtitleFiles, setSubtitleFiles] = useState({});
    // ASS url -> the file's text once downloaded, and the ones the renderer failed on
    const [assFiles, setAssFiles] = useState({});
    const [assFailed, setAssFailed] = useState({});
    // URLs of the fonts attached to the video, for ASS
    const [fonts, setFonts] = useState([]);
    // the <video> itself is fullscreen (its own button), so nothing can be drawn over it
    const [videoFullscreen, setVideoFullscreen] = useState(false);
    // index into subtitles, -1 for off
    const [selectedSubtitle, setSelectedSubtitle] = useState(-1);
    // stream index of the subtitle burned into the current stream, -1 for none
    const [burned, setBurned] = useState(-1);
    // what's being switched while a new stream starts, e.g. "Switching audio…"
    const [switching, setSwitching] = useState("");
    const [prefsMessage, setPrefsMessage] = useState("");
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    // bumped by Try again after the stream failed to start
    const [attempt, setAttempt] = useState(0);
    // the video is waiting for data mid-playback (seeking ahead, slow network)
    const [buffering, setBuffering] = useState(false);
    // which load has taken long enough to explain the wait, see the effect below
    const [slowLoad, setSlowLoad] = useState(null);
    // the audio/subtitle panel. Opened by click rather than focus, because opening a
    // select's native picker takes focus away and would close a focus-based dropdown
    const [languagesOpen, setLanguagesOpen] = useState(false);
    const languagesRef = useRef(null);

    // a tap outside the panel or Escape closes it
    useEffect(() => {
        if (!languagesOpen) return
        function onPointerDown(e) {
            if (!languagesRef.current?.contains(e.target)) setLanguagesOpen(false)
        }
        function onKeyDown(e) {
            if (e.key === "Escape") setLanguagesOpen(false)
        }
        document.addEventListener("pointerdown", onPointerDown)
        document.addEventListener("keydown", onKeyDown)
        return () => {
            document.removeEventListener("pointerdown", onPointerDown)
            document.removeEventListener("keydown", onKeyDown)
        }
    }, [languagesOpen])
    const initialPosition = location.state?.position ?? 0;
    const animeId = location.state?.animeId ?? null;
    const episodeList = location.state?.episodes ?? [];
    const currentIndex = episodeList.findIndex(ep => ep.id === id);
    const nextEpisode = episodeList[currentIndex + 1] ?? null;
    const prevEpisode = episodeList[currentIndex - 1] ?? null;
    const saveIntervalRef = useRef(null);
    const positionRef = useRef(0);
    const durationRef = useRef(0);
    // where and whether to play when a (new) playlist is attached
    const startAtRef = useRef(initialPosition);
    const autoplayRef = useRef(true);
    // hls.js turns every subtitle track off while it swaps streams; don't treat that as the user's choice
    const ignoreTrackChangesRef = useRef(false);

    const baseURL = import.meta.env.VITE_API_URL

    // returns signed playlist/subtitle URLs, the media token in them expires after ~2 hours.
    // start makes the server wait until the transcode reaches that point, audio picks the
    // track (otherwise the user's preferred language). keepSubtitle keeps the current choice.
    // burn is the image subtitle to burn into the video (-1 none), otherwise the server burns
    // one in when the user's preferences pick it
    function fetchStream(start = 0, audio = null, keepSubtitle = undefined, burn = undefined) {
        const params = {}
        if (start > 0) params.start = Math.floor(start)
        if (audio !== null) params.audio = audio
        if (burn !== undefined) params.burn = burn

        return api.get(`/stream/media/${type}/${id}`, { params })
            .then(res => {
                const tracks = (res.data.subtitles ?? []).map(t => ({
                    index: t.index,
                    url: t.url ? `${baseURL}${t.url}` : null,
                    assUrl: t.ass_url ? `${baseURL}${t.ass_url}` : null,
                    burn: t.burn,
                    label: t.label,
                    language: t.language,
                }))
                setAudioTracks(res.data.audio ?? [])
                setSubtitles(tracks)
                setBurned(res.data.burned ?? -1)
                setFonts((res.data.fonts ?? []).map(f => `${baseURL}${f}`))

                if (keepSubtitle === undefined) {
                    setSelectedSubtitle((res.data.subtitles ?? []).findIndex(t => t.default))
                } else {
                    setSelectedSubtitle(keepSubtitle === null ? -1 : tracks.findIndex(t => t.label === keepSubtitle))
                }
                return `${baseURL}${res.data.playlist}`
            })
    }

    function currentSubtitleLabel() {
        return selectedSubtitle >= 0 ? subtitles[selectedSubtitle]?.label ?? null : null
    }

    // the current choices, for the token remint handler which is set up once per playlist
    const choicesRef = useRef({ audio: null, subtitle: null, burn: -1 })
    useEffect(() => {
        choicesRef.current = {
            audio: audioTracks.find(a => a.selected)?.index ?? null,
            subtitle: selectedSubtitle >= 0 ? subtitles[selectedSubtitle]?.label ?? null : null,
            burn: burned,
        }
    }, [audioTracks, subtitles, selectedSubtitle, burned])

    useEffect(() => {
        fetchStream(initialPosition)
            .then(setPlaylistUrl)
            .catch(err => setError(err))
            .finally(() => setLoading(false))
    // eslint-disable-next-line react-hooks/exhaustive-deps -- fetches once per title and retry, the start position is only for the first load
    }, [id, attempt])

    function retry() {
        setError(null)
        setLoading(true)
        setAttempt(a => a + 1)
    }

    // a first play waits for the server to start transcoding, which can take a while:
    // after a few seconds say so, so the spinner doesn't look stuck
    const loadKey = `${id}:${attempt}`
    useEffect(() => {
        if (!loading) return
        const timer = setTimeout(() => setSlowLoad(loadKey), 6000)
        return () => clearTimeout(timer)
    }, [loading, loadKey])

    useEffect(() => {
        const video = videoRef.current
        if (!video) return
        const waiting = () => setBuffering(true)
        const ready = () => setBuffering(false)
        video.addEventListener("waiting", waiting)
        for (const event of ["playing", "canplay", "pause", "emptied"]) video.addEventListener(event, ready)
        return () => {
            video.removeEventListener("waiting", waiting)
            for (const event of ["playing", "canplay", "pause", "emptied"]) video.removeEventListener(event, ready)
        }
    }, [])

    // set up video after playlist loads
    useEffect(() => {
        if (!playlistUrl || !videoRef.current) return;

        const video = videoRef.current;

        if (Hls.isSupported()) {
            // the playlist lists every segment before ffmpeg writes them. Asking for one it
            // hasn't reached (a seek ahead) makes the server restart it there and hold the
            // request open for up to 30s, so wait longer than the default 10s for a first byte
            const start = startAtRef.current
            const hls = new Hls({
                startPosition: start > 0 ? start : 0,
                fragLoadPolicy: {
                    default: {
                        maxTimeToFirstByteMs: 40000,
                        maxLoadTimeMs: 120000,
                        timeoutRetry: { maxNumRetry: 2, retryDelayMs: 0, maxRetryDelayMs: 0 },
                        errorRetry: { maxNumRetry: 6, retryDelayMs: 1000, maxRetryDelayMs: 8000 },
                    },
                },
            })
            // one remint per expiry, cleared once a fragment loads with the new token
            let reminted = false
            let resume = null
            let destroyed = false

            hls.loadSource(playlistUrl)
            hls.attachMedia(video)
            hls.on(Hls.Events.MANIFEST_PARSED, () => {
                ignoreTrackChangesRef.current = false
                setSwitching("")
                // reloaded with a fresh token, put playback back where it was
                if (resume) {
                    video.currentTime = resume.position
                    if (resume.playing) video.play().catch(() => {})
                    resume = null
                    return
                }

                if (autoplayRef.current) video.play().catch(() => {})
            })
            hls.on(Hls.Events.FRAG_LOADED, () => {
                reminted = false
            })
            hls.on(Hls.Events.ERROR, (_, data) => {
                const status = data.response?.code ?? data.networkDetails?.status
                if (status !== 401) return
                if (reminted) {
                    setError("Playback session expired")
                    return
                }
                reminted = true
                resume = { position: video.currentTime, playing: !video.paused }
                const { audio, subtitle, burn } = choicesRef.current
                fetchStream(video.currentTime, audio, subtitle, burn)
                    .then(url => {
                        if (!destroyed) hls.loadSource(url)
                    })
                    .catch(err => setError(err))
            })
            return () => {
                destroyed = true
                ignoreTrackChangesRef.current = true
                hls.destroy()
                video.src = ""
            }
        }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- the remint handler is set up once per playlist, fetchStream reads type and id only
    }, [playlistUrl])

    /**
     * Restart playback from the current position with another audio track or burned in
     * subtitle. Each combination is its own transcode on the server. Returns whether it worked
     */
    function restartStream(audio, burn, keepSubtitle, message) {
        const video = videoRef.current
        if (!video) return Promise.resolve(false)
        startAtRef.current = video.currentTime
        autoplayRef.current = !video.paused
        ignoreTrackChangesRef.current = true
        setSwitching(message)
        setPrefsMessage("")

        return fetchStream(video.currentTime, audio, keepSubtitle, burn)
            .then(url => {
                setPlaylistUrl(url)
                return true
            })
            .catch(err => {
                // the current stream keeps playing, so this doesn't need to stop playback
                setSwitching("")
                ignoreTrackChangesRef.current = false
                toast.error(err, message === "Switching audio…" ? "Could not switch the audio track" : "Could not change the subtitles")
                return false
            })
    }

    function switchAudio(index) {
        restartStream(index, burned, currentSubtitleLabel(), "Switching audio…")
    }

    /**
     * Show a subtitle track (-1 for off). Text tracks switch straight away, an image track
     * has to be burned into the video, so going to or from one restarts the stream
     */
    function chooseSubtitle(i) {
        const previous = selectedSubtitle
        const track = subtitles[i]
        const burn = track?.burn ? track.index : -1
        setSelectedSubtitle(i)
        setPrefsMessage("")
        if (burn === burned) return

        const audio = audioTracks.find(a => a.selected)?.index ?? null
        restartStream(audio, burn, track?.label ?? null, "Switching subtitles…")
            .then(ok => { if (!ok) setSelectedSubtitle(previous) })
    }

    /**
     * Save the languages of the current audio and subtitle tracks as the user's defaults
     */
    function saveAsDefault() {
        const audioLanguage = audioTracks.find(a => a.selected)?.language
        const subtitle = selectedSubtitle >= 0 ? subtitles[selectedSubtitle] : null
        const isCode = lang => /^[a-z]{3}$/.test(lang ?? "") && lang !== "und"

        if (subtitle && !isCode(subtitle.language)) {
            setPrefsMessage("This subtitle track has no language set")
            return
        }

        api.put("/me/preferences", {
            audio_language: isCode(audioLanguage) ? audioLanguage : "default",
            subtitle_language: subtitle ? subtitle.language : "off",
        })
            .then(() => setPrefsMessage("Saved as your default"))
            .catch(err => setPrefsMessage(typeof err.response?.data === "string" ? err.response.data : "Could not save"))
    }

    // subtitle files are written a few seconds after the playlist is ready on a first play.
    // Download them here, retrying until they exist, and only render a <track> once its file
    // is here: a <track> that fails to load never retries, even if it gets a new src later.
    // ASS files are kept as text for the renderer
    useEffect(() => {
        const controller = new AbortController()
        const created = []

        async function download(url, done) {
            for (let attempt = 0; attempt < 30 && !controller.signal.aborted; attempt++) {
                try {
                    const res = await fetch(url, { signal: controller.signal })
                    if (res.ok) {
                        done(await res.text())
                        return
                    }
                } catch {
                    if (controller.signal.aborted) return
                }
                await new Promise(r => setTimeout(r, 2000))
            }
        }

        subtitles.forEach(t => {
            if (t.url) download(t.url, text => {
                const blobUrl = URL.createObjectURL(new Blob([text], { type: "text/vtt" }))
                created.push(blobUrl)
                setSubtitleFiles(files => ({ ...files, [t.url]: blobUrl }))
            })
            if (t.assUrl) download(t.assUrl, text => setAssFiles(files => ({ ...files, [t.assUrl]: text })))
        })

        return () => {
            controller.abort()
            created.forEach(u => URL.revokeObjectURL(u))
        }
    }, [subtitles])

    // how the chosen track is shown: burned in, rendered from its ASS file, or as a <track>
    const selectedTrack = subtitles[selectedSubtitle] ?? null
    const mode = subtitleMode(selectedTrack, {
        assReady: !!assFiles[selectedTrack?.assUrl],
        assFailed: !!assFailed[selectedTrack?.assUrl],
        videoFullscreen,
    })
    const assCanvasRef = useRef(null)
    const assContent = mode === "ass" ? assFiles[selectedTrack.assUrl] : null
    useAssRenderer(videoRef, assCanvasRef, assContent, fonts, err => {
        console.warn("ASS subtitles could not be shown, using plain ones", err)
        setAssFailed(failed => ({ ...failed, [selectedTrack?.assUrl]: true }))
    })

    useEffect(() => {
        const onChange = () => setVideoFullscreen(!!videoRef.current && document.fullscreenElement === videoRef.current)
        document.addEventListener("fullscreenchange", onChange)
        return () => document.removeEventListener("fullscreenchange", onChange)
    }, [])

    // show only the selected subtitle track, when it's shown as a <track>. <track> elements only
    // exist for downloaded files, so match them to the menu by url rather than position
    function applySubtitleSelection() {
        const video = videoRef.current
        if (!video) return
        const selectedUrl = mode === "vtt" ? subtitles[selectedSubtitle]?.url : null
        video.querySelectorAll("track").forEach(el => {
            const trackMode = el.dataset.url === selectedUrl ? "showing" : "disabled"
            if (el.track.mode !== trackMode) el.track.mode = trackMode
        })
    }

    useEffect(applySubtitleSelection, [subtitles, subtitleFiles, selectedSubtitle, mode])

    // keep the menu in sync when the browser's own captions button is used. The browser also
    // turns tracks on by itself as they're added, so only a single different track (or none)
    // counts as the user's choice; anything else puts the menu's choice back
    useEffect(() => {
        const video = videoRef.current
        if (!video) return
        const textTracks = video.textTracks
        function onChange() {
            if (ignoreTrackChangesRef.current) return
            // a burned in or ASS track isn't one of the browser's, so its captions menu can't
            // tell what's showing: keep the menu's choice
            if (mode === "burn" || mode === "ass") {
                applySubtitleSelection()
                return
            }
            const elements = Array.from(video.querySelectorAll("track"))
            const showing = elements.filter(el => el.track.mode === "showing")
            const selectedUrl = subtitles[selectedSubtitle]?.url
            const selectedElement = elements.find(el => el.dataset.url === selectedUrl)

            if (selectedUrl && !selectedElement) {
                // the chosen track is still downloading
                applySubtitleSelection()
            } else if (showing.length === 1 && showing[0] !== selectedElement) {
                chooseSubtitle(subtitles.findIndex(t => t.url === showing[0].dataset.url))
            } else if (showing.length === 0 && selectedElement) {
                chooseSubtitle(-1)
            } else if (showing.length > 1) {
                applySubtitleSelection()
            }
        }
        textTracks.addEventListener("change", onChange)
        return () => textTracks.removeEventListener("change", onChange)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- applySubtitleSelection and chooseSubtitle read the same state listed here
    }, [loading, subtitles, subtitleFiles, selectedSubtitle, burned, audioTracks, mode])

    // save progress every 5 seconds, on pause, when the page is hidden or closed, and on
    // unmount, with the seconds actually watched since the last save (for Insights)
    useEffect(() => {
        const video = videoRef.current
        let last = null
        let watched = 0

        function onTimeUpdate() {
            if (!video.paused && !video.seeking) watched += watchedBetween(last, video.currentTime)
            last = video.currentTime
        }

        function updateRefs() {
            if (video) {
                positionRef.current = video.currentTime
                durationRef.current = video.duration || durationRef.current
            }
        }

        // keepalive lets the request finish while the page is closing
        function saveProgress(keepalive = false) {
            updateRefs()
            if (positionRef.current <= 0) return
            const endpoint = type === "movie" ? `/movies/${id}/progress` : `/episodes/${id}/progress`
            const sent = watched
            watched = 0
            api.put(endpoint, {
                position_secs: positionRef.current,
                duration_secs: durationRef.current,
                watched_secs: sent,
            }, keepalive ? { adapter: "fetch", fetchOptions: { keepalive: true } } : undefined)
                .catch(() => { watched += sent })
        }

        const onPause = () => saveProgress()
        const onVisibility = () => { if (document.visibilityState === "hidden") saveProgress(true) }
        const onPageHide = () => saveProgress(true)
        video?.addEventListener("timeupdate", onTimeUpdate)
        video?.addEventListener("pause", onPause)
        document.addEventListener("visibilitychange", onVisibility)
        window.addEventListener("pagehide", onPageHide)
        saveIntervalRef.current = setInterval(saveProgress, 5000)

        return () => {
            clearInterval(saveIntervalRef.current)
            video?.removeEventListener("timeupdate", onTimeUpdate)
            video?.removeEventListener("pause", onPause)
            document.removeEventListener("visibilitychange", onVisibility)
            window.removeEventListener("pagehide", onPageHide)
            saveProgress(true)
        }
    }, [type, id])

    function goToEpisode(ep) {
        setLoading(true)
        switchViewer(navigate, location, `/watch/episode/${ep.id}`, { position: 0 });
    }

    function exit() {
        exitViewer(navigate, location, type === "movie" ? `/movies/${id}` : animeId ? `/anime/${animeId}` : null)
    }

    // when an episode ends the next one starts after a countdown, unless it's cancelled or the
    // episode is played again. Tied to the episode that ended, so moving on clears it
    const [upNext, setUpNext] = useState(null)
    const countdown = upNext?.from === id ? upNext.secs : null
    useEffect(() => {
        const video = videoRef.current
        if (!video || !nextEpisode) return
        const start = () => setUpNext({ from: id, secs: UP_NEXT_SECS })
        const cancel = () => setUpNext(null)
        video.addEventListener("ended", start)
        video.addEventListener("play", cancel)
        video.addEventListener("seeking", cancel)
        return () => {
            video.removeEventListener("ended", start)
            video.removeEventListener("play", cancel)
            video.removeEventListener("seeking", cancel)
        }
    }, [id, nextEpisode])
    useEffect(() => {
        if (countdown === null) return
        const timer = setTimeout(() => {
            if (countdown <= 1) goToEpisode(nextEpisode)
            else setUpNext({ from: id, secs: countdown - 1 })
        }, 1000)
        return () => clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- each second of the countdown sets the next one going
    }, [countdown])

    const playingEpisode = episodeList[currentIndex]
    useDocumentTitle(playingEpisode ? playingEpisode.title ?? `Episode ${playingEpisode.episode_number}` : null)

    // Space or K plays and pauses, the arrows skip, F is fullscreen, M mutes, Shift+N is the next episode
    const playerRef = useRef(null)
    useEffect(() => {
        function onKeyDown(e) {
            const video = videoRef.current
            if (!video || keyForFocused(e)) return
            const key = e.key.length === 1 ? e.key.toLowerCase() : e.key
            if (key === " " || key === "k") {
                if (video.paused) video.play().catch(() => {})
                else video.pause()
            } else if (key === "ArrowLeft" || key === "ArrowRight") {
                const to = video.currentTime + (key === "ArrowLeft" ? -SKIP_SECS : SKIP_SECS)
                video.currentTime = Math.min(Math.max(to, 0), video.duration || to)
            } else if (key === "f") {
                if (document.fullscreenElement) document.exitFullscreen().catch(() => {})
                else playerRef.current?.requestFullscreen?.().catch(() => {})
            } else if (key === "m") {
                video.muted = !video.muted
            } else if (key === "n" && e.shiftKey && nextEpisode) {
                goToEpisode(nextEpisode)
            } else {
                return
            }
            e.preventDefault()
        }
        document.addEventListener("keydown", onKeyDown)
        return () => document.removeEventListener("keydown", onKeyDown)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- goToEpisode only needs the episode it's given
    }, [nextEpisode])

    if (error) return <div className="min-h-dvh bg-base-100">
        <ErrorState error={error} title="Couldn't play this video" onRetry={retry} showBack />
    </div>

    const selectedAudio = audioTracks.find(a => a.selected)
    const currentEpisode = episodeList[currentIndex]

    return (
    <div ref={playerRef} className="flex flex-col h-dvh bg-black text-white">
        <header className="relative z-20 flex items-center gap-2 px-2 sm:px-4 pt-[env(safe-area-inset-top)] h-[calc(3.5rem+env(safe-area-inset-top))] bg-linear-to-b from-black/90 to-black/60 backdrop-blur border-b border-white/5">
            <button className="btn btn-ghost btn-sm gap-1.5 text-white hover:bg-white/10" onClick={exit} aria-label="Back">
                <ArrowLeft size={18}/> <span className="hidden sm:inline">Back</span>
            </button>
            <div className="flex-1 min-w-0 text-center">
                {currentEpisode && <>
                    <p className="text-sm font-medium truncate">{currentEpisode.title ?? `Episode ${currentEpisode.episode_number}`}</p>
                    <p className="text-xs text-white/50 tabular-nums">
                        {currentEpisode.season_number ? `Season ${currentEpisode.season_number} · ` : ""}Episode {currentEpisode.episode_number}
                    </p>
                </>}
            </div>
            <div className="flex items-center gap-1">
                {episodeList.length > 0 && <button
                    className="btn btn-ghost btn-sm btn-square text-white hover:bg-white/10"
                    disabled={!prevEpisode}
                    onClick={() => prevEpisode && goToEpisode(prevEpisode)}
                    aria-label="Previous episode"
                >
                    <SkipBack size={18}/>
                </button>}
                {episodeList.length > 0 && <button
                    className="btn btn-ghost btn-sm text-white hover:bg-white/10 gap-1.5"
                    disabled={!nextEpisode}
                    onClick={() => nextEpisode && goToEpisode(nextEpisode)}
                    aria-label="Next episode"
                >
                    <span className="hidden sm:inline">Next</span> <SkipForward size={18}/>
                </button>}
                <div className="relative" ref={languagesRef}>
                    <button
                        className={`btn btn-ghost btn-sm gap-1.5 text-white hover:bg-white/10 ${languagesOpen ? "bg-white/10" : ""}`}
                        aria-label="Audio and subtitles"
                        aria-expanded={languagesOpen}
                        onClick={() => setLanguagesOpen(!languagesOpen)}
                    >
                        <Languages size={18}/> <span className="hidden md:inline">Languages</span>
                    </button>
                    {languagesOpen && <div className="absolute right-0 top-full surface-raised text-base-content z-30 mt-2 w-[min(20rem,calc(100vw-1rem))] p-4 flex flex-col gap-4 animate-fade-in">
                        <label className="flex flex-col gap-1.5">
                            <span className="text-xs font-medium uppercase tracking-wider text-base-content/50">Audio</span>
                            <select
                                className="select select-sm w-full"
                                value={selectedAudio?.index ?? ""}
                                disabled={audioTracks.length < 2 || switching !== ""}
                                onChange={e => switchAudio(Number(e.target.value))}
                            >
                                {audioTracks.length === 0 && <option value="">No audio</option>}
                                {audioTracks.map(a => (
                                    <option key={a.index} value={a.index}>{a.label}</option>
                                ))}
                            </select>
                        </label>
                        <label className="flex flex-col gap-1.5">
                            <span className="text-xs font-medium uppercase tracking-wider text-base-content/50">Subtitles</span>
                            <select
                                className="select select-sm w-full"
                                value={selectedSubtitle}
                                disabled={switching !== ""}
                                onChange={e => chooseSubtitle(Number(e.target.value))}
                            >
                                <option value={-1}>Off</option>
                                {subtitles.map((t, i) => (
                                    <option key={t.index} value={i}>{t.burn ? `${t.label} (burned in)` : t.label}</option>
                                ))}
                            </select>
                            {subtitles.length === 0 && (
                                <span className="text-xs text-base-content/60">This video has no subtitles</span>
                            )}
                        </label>
                        {switching && <span className="text-xs text-base-content/60 flex items-center gap-2">
                            <span className="loading loading-spinner loading-xs"></span> {switching}
                        </span>}
                        <button className="btn btn-sm btn-primary" onClick={saveAsDefault}>
                            Use these languages by default
                        </button>
                        {prefsMessage && <span className="text-xs text-base-content/70">{prefsMessage}</span>}
                    </div>}
                </div>
            </div>
        </header>
        <div className="relative flex flex-1 min-h-0 items-center justify-center">
            {/* the video element stays mounted while loading so hls.js always has it to attach to */}
            {loading && (
                <div className="absolute inset-0 z-10 flex flex-col items-center justify-center gap-4 text-center px-6" role="status">
                    <span className="loading loading-ring loading-lg text-primary"></span>
                    <p className="font-medium">Preparing the video…</p>
                    {slowLoad === loadKey && (
                        <p className="text-sm text-white/60 max-w-sm animate-fade-in">
                            The first play of a video takes a little longer while the server gets it ready.
                        </p>
                    )}
                </div>
            )}
            {!loading && buffering && (
                <div className="absolute inset-0 z-10 flex items-center justify-center pointer-events-none" role="status" aria-label="Buffering">
                    <span className="grid size-20 place-items-center rounded-full bg-black/50 backdrop-blur">
                        <span className="loading loading-spinner loading-lg"></span>
                    </span>
                </div>
            )}
            <video
                ref={videoRef}
                controls
                playsInline
                crossOrigin="anonymous"
                className={`w-full h-full max-h-full object-contain ${loading ? "invisible" : ""}`}
            >
                {subtitles.filter(t => t.url && subtitleFiles[t.url]).map(t => (
                    <track
                        key={t.url}
                        data-url={t.url}
                        kind="subtitles"
                        src={subtitleFiles[t.url]}
                        srcLang={t.language || "und"}
                        label={t.label}
                    />
                ))}
            </video>
            {/* a new canvas for each file, a canvas can only be given to one renderer */}
            {assContent && <canvas key={selectedTrack.assUrl} ref={assCanvasRef} className="absolute pointer-events-none" aria-hidden="true"/>}
            {countdown !== null && nextEpisode && (
                <div className="absolute z-20 right-3 sm:right-6 bottom-16 sm:bottom-20 left-3 sm:left-auto sm:w-80 surface-raised text-base-content p-4 flex items-center gap-4 animate-fade-up" role="status">
                    <div className="radial-progress text-primary shrink-0 text-sm font-semibold tabular-nums"
                        style={{ "--value": (countdown / UP_NEXT_SECS) * 100, "--size": "3rem", "--thickness": "3px" }}
                        aria-label={`Next episode in ${countdown} seconds`}>
                        {countdown}
                    </div>
                    <div className="min-w-0 flex-1">
                        <p className="text-xs font-medium uppercase tracking-wider text-primary">Up next</p>
                        <p className="text-sm font-medium truncate">
                            Episode {nextEpisode.episode_number}{nextEpisode.title ? ` · ${nextEpisode.title}` : ""}
                        </p>
                        <button className="btn btn-primary btn-sm mt-2" onClick={() => goToEpisode(nextEpisode)}>Play now</button>
                    </div>
                    <button className="btn btn-ghost btn-sm btn-circle self-start" onClick={() => setUpNext(null)} aria-label="Cancel next episode">
                        <X size={16}/>
                    </button>
                </div>
            )}
        </div>
    </div>
)
}
