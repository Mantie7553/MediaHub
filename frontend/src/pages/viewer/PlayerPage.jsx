import { useState, useEffect, useRef } from "react"
import { useParams, useLocation, useNavigate } from "react-router-dom"
import Hls from "hls.js"
import api from "../../services/api"
import ErrorState from "../../components/states/ErrorState"
import { toast } from "../../stores/useToastStore"
import { exitViewer, switchViewer } from "../../utils/navigation"
import { keyForFocused } from "../../utils/keys"
import useDocumentTitle from "../../hooks/useDocumentTitle"
import useApi from "../../hooks/useApi"
import useAssRenderer from "../../hooks/useAssRenderer"
import { subtitleMode } from "../../utils/subtitles"
import { watchedBetween } from "../../utils/watchTime"
import { pictureRect, subtitleLift } from "../../utils/player"
import useVideoState from "../../hooks/useVideoState"
import usePlayerChrome from "../../hooks/usePlayerChrome"
import SeekBar from "../../components/player/SeekBar"
import PlayerControls, { CenterControls } from "../../components/player/PlayerControls"
import SubtitleOverlay from "../../components/player/SubtitleOverlay"
import { ArrowLeft, FastForward, Settings, X } from "lucide-react"

// how long the next episode waits after one ends, and how far the arrow keys and skip buttons go
const UP_NEXT_SECS = 5
const SKIP_SECS = 10
const SPEEDS = [0.5, 0.75, 1, 1.25, 1.5, 1.75, 2]
// a skipped part ending this close to the end of the video skips to the end, so up next starts
const SKIP_TO_END_SECS = 2
// what the skip button and the auto skip message call each part
const SKIP_NAMES = { op: "opening", ed: "ending", recap: "recap", preview: "preview" }
const NO_SEGMENTS = []

export default function PlayerPage() {
    const location = useLocation();
    const navigate = useNavigate();
    const { type, id } = useParams();
    const videoRef = useRef(null);
    const playerRef = useRef(null);
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
    // the <video> itself is fullscreen (iPhones, which can't fullscreen anything else), so
    // nothing can be drawn over it
    const [videoFullscreen, setVideoFullscreen] = useState(false);
    // the video is in its own picture-in-picture window, where the page can't draw either
    const [pictureInPicture, setPictureInPicture] = useState(false);
    const videoAlone = videoFullscreen || pictureInPicture;
    // the whole player is fullscreen, with its controls and everything over the video
    const [playerFullscreen, setPlayerFullscreen] = useState(false);
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
    // openings, endings, recaps and previews: [{ type, start, end }], for the video they're of
    const [skipTimes, setSkipTimes] = useState({ for: null, segments: NO_SEGMENTS });
    const skipSegments = skipTimes.for === id ? skipTimes.segments : NO_SEGMENTS;
    // the one playing now, as an index into the segments it was found in
    const [activeSkip, setActiveSkip] = useState({ segments: null, index: -1 });
    const skippable = activeSkip.segments === skipSegments ? skipSegments[activeSkip.index] ?? null : null;
    // the user's preference to skip them without asking
    const { data: prefs } = useApi("/me/preferences");
    const autoSkip = !!prefs?.auto_skip;

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

    // the parts to offer to skip. A video without any (or the request failing) just has no button
    useEffect(() => {
        let cancelled = false
        api.get(`/stream/skip/${type}/${id}`)
            .then(res => { if (!cancelled) setSkipTimes({ for: id, segments: res.data.segments ?? [] }) })
            .catch(() => {})
        return () => { cancelled = true }
    }, [type, id])

    // the video stays mounted under the error, it shouldn't keep playing there
    useEffect(() => {
        if (error) videoRef.current?.pause()
    }, [error])

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
        videoFullscreen: videoAlone,
    })
    const assCanvasRef = useRef(null)
    const assContent = mode === "ass" ? assFiles[selectedTrack.assUrl] : null
    useAssRenderer(videoRef, assCanvasRef, assContent, fonts, err => {
        console.warn("ASS subtitles could not be shown, using plain ones", err)
        setAssFailed(failed => ({ ...failed, [selectedTrack?.assUrl]: true }))
    })

    useEffect(() => {
        const video = videoRef.current
        const onChange = () => setPlayerFullscreen(!!playerRef.current && document.fullscreenElement === playerRef.current)
        const onBegin = () => setVideoFullscreen(true)
        const onEnd = () => setVideoFullscreen(false)
        const onEnterPip = () => setPictureInPicture(true)
        const onLeavePip = () => setPictureInPicture(false)
        // Safari's own picture-in-picture (its button, or leaving the app on an iPhone)
        const onPresentation = () => setPictureInPicture(video.webkitPresentationMode === "picture-in-picture")
        document.addEventListener("fullscreenchange", onChange)
        video?.addEventListener("webkitbeginfullscreen", onBegin)
        video?.addEventListener("webkitendfullscreen", onEnd)
        video?.addEventListener("enterpictureinpicture", onEnterPip)
        video?.addEventListener("leavepictureinpicture", onLeavePip)
        video?.addEventListener("webkitpresentationmodechanged", onPresentation)
        return () => {
            document.removeEventListener("fullscreenchange", onChange)
            video?.removeEventListener("webkitbeginfullscreen", onBegin)
            video?.removeEventListener("webkitendfullscreen", onEnd)
            video?.removeEventListener("enterpictureinpicture", onEnterPip)
            video?.removeEventListener("leavepictureinpicture", onLeavePip)
            video?.removeEventListener("webkitpresentationmodechanged", onPresentation)
        }
    }, [])

    // fullscreen the whole player so the controls, subtitles, skip and up next stay. iPhones
    // can only fullscreen the video, which hands it to their own player
    function toggleFullscreen() {
        const video = videoRef.current
        if (document.fullscreenElement) document.exitFullscreen().catch(() => {})
        else if (playerRef.current?.requestFullscreen) playerRef.current.requestFullscreen().catch(() => {})
        else video?.webkitEnterFullscreen?.()
    }

    // show only the selected subtitle track, when it's shown as a <track>. The player draws it
    // itself (SubtitleOverlay), so it's "hidden" rather than "showing", except when the video is
    // somewhere the page can't draw (the iPhone player, picture in picture), where the browser
    // shows it. <track> elements only exist for downloaded files, so match them to the menu by
    // url rather than position
    function applySubtitleSelection() {
        const video = videoRef.current
        if (!video) return
        const selectedUrl = mode === "vtt" ? subtitles[selectedSubtitle]?.url : null
        const on = videoAlone ? "showing" : "hidden"
        video.querySelectorAll("track").forEach(el => {
            const trackMode = el.dataset.url === selectedUrl ? on : "disabled"
            if (el.track.mode !== trackMode) el.track.mode = trackMode
        })
    }

    useEffect(applySubtitleSelection, [subtitles, subtitleFiles, selectedSubtitle, mode, videoAlone])

    // the browser turns tracks on by itself as they're added: put the menu's choice back. While
    // the iPhone player has the video its captions menu can be used, so keep the menu in sync
    // with it there: only a single different track (or none) counts as the user's choice
    useEffect(() => {
        const video = videoRef.current
        if (!video) return
        const textTracks = video.textTracks
        function onChange() {
            if (ignoreTrackChangesRef.current) return
            // a burned in or ASS track isn't one of the browser's, so its captions menu can't
            // tell what's showing: keep the menu's choice
            if (!videoFullscreen || mode === "burn" || mode === "ass") {
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
    }, [loading, subtitles, subtitleFiles, selectedSubtitle, burned, audioTracks, mode, videoFullscreen, videoAlone])

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

    // jump past a part of the video, to the very end when only a moment would be left
    function skipPast(segment) {
        const video = videoRef.current
        if (!video) return
        const duration = video.duration
        video.currentTime = duration && segment.end >= duration - SKIP_TO_END_SECS ? duration : segment.end
    }

    // follow which part is playing for the skip button. With auto skip on each part is skipped
    // once per video, so going back into one on purpose plays it
    useEffect(() => {
        const video = videoRef.current
        if (!video || skipSegments.length === 0) return
        const skipped = new Set()
        let current = -1
        function onTimeUpdate() {
            const t = video.currentTime
            // nothing is worth skipping in the last second of a part
            const i = skipSegments.findIndex(seg => t >= seg.start && t < seg.end - 1)
            if (i >= 0 && autoSkip && !skipped.has(i) && !video.paused) {
                skipped.add(i)
                skipPast(skipSegments[i])
                toast.success(`Skipped the ${SKIP_NAMES[skipSegments[i].type]}`)
                return
            }
            if (i !== current) {
                current = i
                setActiveSkip({ segments: skipSegments, index: i })
            }
        }
        video.addEventListener("timeupdate", onTimeUpdate)
        return () => video.removeEventListener("timeupdate", onTimeUpdate)
    }, [skipSegments, autoSkip])

    // for the S key, whose handler is set up once per episode
    const skippableRef = useRef(null)
    useEffect(() => { skippableRef.current = skippable }, [skippable])

    const playingEpisode = episodeList[currentIndex]
    useDocumentTitle(playingEpisode ? playingEpisode.title ?? `Episode ${playingEpisode.episode_number}` : null)

    const videoState = useVideoState(videoRef)
    // the seek bar is being dragged, or a mouse is over the controls: they stay up meanwhile
    const [scrubbing, setScrubbing] = useState(false)
    const [overControls, setOverControls] = useState(false)
    const chrome = usePlayerChrome(loading || videoState.paused || languagesOpen || scrubbing || overControls)
    const { show: showChrome, hide: hideChrome } = chrome

    // pausing brings the controls back
    useEffect(() => {
        const video = videoRef.current
        video?.addEventListener("pause", showChrome)
        return () => video?.removeEventListener("pause", showChrome)
    }, [showChrome])

    function togglePlay() {
        const video = videoRef.current
        if (!video) return
        if (video.paused) video.play().catch(() => {})
        else video.pause()
    }

    function seekBy(secs) {
        const video = videoRef.current
        if (!video) return
        const to = video.currentTime + secs
        video.currentTime = Math.min(Math.max(to, 0), video.duration || to)
    }

    // Space or K plays and pauses, the arrows skip, F is fullscreen, M mutes, S skips the opening
    // or ending playing now, Shift+N is the next episode. Each brings the controls up
    useEffect(() => {
        function onKeyDown(e) {
            const video = videoRef.current
            if (!video || keyForFocused(e)) return
            const key = e.key.length === 1 ? e.key.toLowerCase() : e.key
            if (key === " " || key === "k") {
                togglePlay()
            } else if (key === "ArrowLeft" || key === "ArrowRight") {
                seekBy(key === "ArrowLeft" ? -SKIP_SECS : SKIP_SECS)
            } else if (key === "f") {
                toggleFullscreen()
            } else if (key === "m") {
                video.muted = !video.muted
            } else if (key === "s" && !e.shiftKey && skippableRef.current) {
                skipPast(skippableRef.current)
            } else if (key === "n" && e.shiftKey && nextEpisode) {
                goToEpisode(nextEpisode)
            } else {
                return
            }
            e.preventDefault()
            showChrome()
        }
        document.addEventListener("keydown", onKeyDown)
        return () => document.removeEventListener("keydown", onKeyDown)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- goToEpisode only needs the episode it's given, the others read refs
    }, [nextEpisode, showChrome])

    // the player's size and the bottom controls' height, to keep subtitles clear of the controls
    const bottomBarRef = useRef(null)
    const [box, setBox] = useState({ width: 0, height: 0 })
    const [barHeight, setBarHeight] = useState(0)
    useEffect(() => {
        const video = videoRef.current
        const bar = bottomBarRef.current
        if (!video || !bar) return
        const observer = new ResizeObserver(() => {
            setBox({ width: video.clientWidth, height: video.clientHeight })
            setBarHeight(bar.offsetHeight)
        })
        observer.observe(video)
        observer.observe(bar)
        return () => observer.disconnect()
    }, [])

    // a tap on the video shows the controls or, when they're up, hides them. A mouse click plays
    // and pauses instead, as moving the mouse already shows them. Taps on the controls are theirs,
    // and a tap that closes the settings menu does only that
    const menuOpenAtPressRef = useRef(false)
    function onPlayerPointerDown(e) {
        menuOpenAtPressRef.current = languagesOpen
        if (e.target.closest("[data-controls]")) showChrome()
    }
    function onPlayerPointerUp(e) {
        if (e.target.closest("[data-controls]") || menuOpenAtPressRef.current || e.button !== 0) return
        if (e.pointerType === "mouse") {
            if (!loading) togglePlay()
        } else if (chrome.visible) hideChrome()
        else showChrome()
    }
    function onPlayerPointerMove(e) {
        if (e.pointerType === "mouse") showChrome()
    }
    function onPlayerDoubleClick(e) {
        if (!e.target.closest("[data-controls]")) toggleFullscreen()
    }
    // a mouse over the controls keeps them up
    const controlsHover = {
        onPointerEnter: e => { if (e.pointerType === "mouse") setOverControls(true) },
        onPointerLeave: e => { if (e.pointerType === "mouse") setOverControls(false) },
    }

    const selectedAudio = audioTracks.find(a => a.selected)
    const currentEpisode = episodeList[currentIndex]
    const fullscreen = playerFullscreen || videoFullscreen
    // picture in picture is for phones and tablets only, the browser's own one is turned off on
    // a computer (Chrome and Edge show no subtitles in its window)
    const touchScreen = window.matchMedia("(pointer: coarse)").matches
    // while loading only the top bar shows, so Back can always be reached
    const controlsUp = chrome.visible && !loading && !error
    const picture = pictureRect(box, videoState.videoWidth, videoState.videoHeight)
    const lift = controlsUp ? subtitleLift(barHeight, box, videoState.videoWidth, videoState.videoHeight) : 0
    // skip and up next sit just above the bottom controls while they're up
    const overlayBottom = controlsUp ? barHeight + 12 : 24

    const actions = {
        togglePlay,
        skip: seekBy,
        prev: prevEpisode ? () => goToEpisode(prevEpisode) : null,
        next: nextEpisode ? () => goToEpisode(nextEpisode) : null,
        toggleMute() {
            const video = videoRef.current
            if (!video) return
            if (video.muted && video.volume === 0) video.volume = 0.5
            video.muted = !video.muted
        },
        setVolume(volume) {
            const video = videoRef.current
            if (!video) return
            video.volume = volume
            video.muted = volume === 0
        },
        toggleFullscreen,
        togglePictureInPicture() {
            if (document.pictureInPictureElement) document.exitPictureInPicture().catch(() => {})
            else videoRef.current?.requestPictureInPicture?.().catch(() => {})
        },
    }

    // the default rate too, so a new stream (another audio track) keeps the speed
    function setSpeed(rate) {
        const video = videoRef.current
        if (!video) return
        video.defaultPlaybackRate = rate
        video.playbackRate = rate
    }

    return (
    <div
        ref={playerRef}
        className={`relative h-dvh overflow-hidden bg-black text-white select-none ${!chrome.visible && !loading ? "cursor-none" : ""}`}
        onPointerDown={onPlayerPointerDown}
        onPointerUp={onPlayerPointerUp}
        onPointerMove={onPlayerPointerMove}
        onDoubleClick={onPlayerDoubleClick}
    >
        {/* the video element stays mounted while loading and on an error, so hls.js and the
            listeners always have it */}
        <video
            ref={videoRef}
            playsInline
            disablePictureInPicture={!touchScreen}
            crossOrigin="anonymous"
            className={`absolute inset-0 w-full h-full object-contain ${loading ? "invisible" : ""}`}
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
        {/* a new canvas for each file, a canvas can only be given to one renderer. The wrapper
            moves it up with the controls without touching how the renderer places it */}
        {assContent && <div className="absolute inset-0 pointer-events-none transition-transform duration-200 ease-out" style={{ transform: `translateY(${-lift}px)` }}>
            <canvas key={selectedTrack.assUrl} ref={assCanvasRef} className="absolute pointer-events-none" aria-hidden="true"/>
        </div>}
        {mode === "vtt" && !videoAlone && (
            <SubtitleOverlay
                videoRef={videoRef}
                trackUrl={selectedTrack?.url ?? null}
                trackReady={!!subtitleFiles[selectedTrack?.url]}
                picture={picture}
                lift={lift}
            />
        )}
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
        {/* phones get the play and skip buttons in the middle, where a thumb reaches them */}
        <div className="sm:hidden absolute inset-0 z-10 flex items-center justify-center pointer-events-none">
            <div
                data-controls
                className={`transition-opacity duration-200 ${controlsUp && !buffering ? "opacity-100 pointer-events-auto" : "opacity-0"}`}
                aria-hidden={!controlsUp}
            >
                <CenterControls video={videoState} skipSecs={SKIP_SECS} actions={actions} hasEpisodes={episodeList.length > 0}/>
            </div>
        </div>
        <header className={`absolute inset-x-0 top-0 z-20 pointer-events-none bg-linear-to-b from-black/80 via-black/40 to-transparent pb-8 pt-[env(safe-area-inset-top)] transition-all duration-200 ease-out ${chrome.visible || loading ? "opacity-100 translate-y-0" : "opacity-0 -translate-y-2"}`}>
            <div data-controls {...controlsHover} className={`flex items-center gap-2 h-14 px-2 sm:px-4 ${chrome.visible || loading ? "pointer-events-auto" : ""}`}>
                <button className="btn btn-ghost btn-sm gap-1.5 text-white hover:bg-white/10 border-0 shrink-0" onClick={exit} aria-label="Back">
                    <ArrowLeft size={18}/> <span className="hidden sm:inline">Back</span>
                </button>
                <div className="flex-1 min-w-0 text-center [text-shadow:0_1px_3px_rgb(0_0_0/0.6)]">
                    {currentEpisode && <>
                        <p className="text-sm font-medium truncate">{currentEpisode.title ?? `Episode ${currentEpisode.episode_number}`}</p>
                        <p className="text-xs text-white/60 tabular-nums">
                            {currentEpisode.season_number ? `Season ${currentEpisode.season_number} · ` : ""}Episode {currentEpisode.episode_number}
                        </p>
                    </>}
                </div>
                {/* as wide as Back, so the title stays centred */}
                <div className="w-8 sm:w-20 shrink-0"></div>
            </div>
        </header>
        {skippable && !loading && countdown === null && (
            <button
                data-controls
                className="absolute z-20 right-3 sm:right-6 btn border-white/20 bg-black/60 hover:bg-black/80 text-white backdrop-blur gap-2 animate-fade-up transition-[bottom] duration-200 ease-out"
                style={{ bottom: overlayBottom }}
                onClick={() => skipPast(skippable)}
            >
                <FastForward size={16}/> Skip {SKIP_NAMES[skippable.type]}
            </button>
        )}
        {countdown !== null && nextEpisode && (
            <div
                data-controls
                className="absolute z-20 right-3 sm:right-6 left-3 sm:left-auto sm:w-80 surface-raised text-base-content p-4 flex items-center gap-4 animate-fade-up transition-[bottom] duration-200 ease-out"
                style={{ bottom: overlayBottom }}
                role="status"
            >
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
        {/* burned in subtitles are part of the picture and can't move, so the shade is lighter over them */}
        <div className={`absolute inset-x-0 bottom-0 z-20 pointer-events-none bg-linear-to-t ${mode === "burn" ? "from-black/55 via-black/20" : "from-black/85 via-black/45"} to-transparent pt-12 transition-all duration-200 ease-out ${controlsUp ? "opacity-100 translate-y-0" : "opacity-0 translate-y-2"}`}>
            <div
                ref={bottomBarRef}
                data-controls
                {...controlsHover}
                className={`flex flex-col px-3 sm:px-5 pb-[max(0.5rem,env(safe-area-inset-bottom))] ${controlsUp ? "pointer-events-auto" : ""}`}
            >
                <SeekBar
                    currentTime={videoState.currentTime}
                    duration={videoState.duration}
                    buffered={videoState.buffered}
                    segments={skipSegments}
                    onSeek={t => { if (videoRef.current) videoRef.current.currentTime = t }}
                    onScrubbing={setScrubbing}
                />
                <PlayerControls video={videoState} skipSecs={SKIP_SECS} actions={actions} hasEpisodes={episodeList.length > 0} fullscreen={fullscreen}>
                    <div className="relative" ref={languagesRef}>
                        <button
                            className={`btn btn-ghost btn-sm btn-square text-white hover:bg-white/10 border-0 ${languagesOpen ? "bg-white/10" : ""}`}
                            aria-label="Audio, subtitles and speed"
                            aria-expanded={languagesOpen}
                            onClick={() => setLanguagesOpen(!languagesOpen)}
                        >
                            <Settings size={18} className={`transition-transform duration-300 ${languagesOpen ? "rotate-45" : ""}`}/>
                        </button>
                        {languagesOpen && <div className="absolute right-0 bottom-full mb-3 surface-raised text-base-content z-30 w-[min(20rem,calc(100vw-1.5rem))] max-h-[calc(100dvh-8rem)] overflow-y-auto p-4 flex flex-col gap-4 animate-fade-in select-text">
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
                            <label className="flex flex-col gap-1.5 border-t border-white/5 pt-4">
                                <span className="text-xs font-medium uppercase tracking-wider text-base-content/50">Speed</span>
                                <select
                                    className="select select-sm w-full"
                                    value={videoState.rate}
                                    onChange={e => setSpeed(Number(e.target.value))}
                                >
                                    {!SPEEDS.includes(videoState.rate) && <option value={videoState.rate}>{videoState.rate}×</option>}
                                    {SPEEDS.map(s => <option key={s} value={s}>{s === 1 ? "Normal" : `${s}×`}</option>)}
                                </select>
                            </label>
                        </div>}
                    </div>
                </PlayerControls>
            </div>
        </div>
        {error && (
            <div data-controls className="absolute inset-0 z-40 overflow-y-auto bg-base-100 text-base-content select-text">
                <ErrorState error={error} title="Couldn't play this video" onRetry={retry} showBack />
            </div>
        )}
    </div>
)
}
