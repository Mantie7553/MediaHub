import { useState, useEffect, useRef } from "react"
import { useParams, useLocation, useNavigate } from "react-router-dom"
import Hls from "hls.js"
import api from "../../services/api"
import Loading from "../../components/states/Loading"
import Error from "../../components/states/Error"
import { ArrowLeft, ArrowRight, Languages } from "lucide-react"

export default function PlayerPage() {
    const location = useLocation();
    const navigate = useNavigate();
    const { type, id } = useParams();
    const videoRef = useRef(null);
    const [playlistUrl, setPlaylistUrl] = useState(null);
    // [{ index, label, language, selected }]
    const [audioTracks, setAudioTracks] = useState([]);
    // [{ url, label, language }]
    const [subtitles, setSubtitles] = useState([]);
    // subtitle url -> downloaded file (object URL) once it exists
    const [subtitleFiles, setSubtitleFiles] = useState({});
    // index into subtitles, -1 for off
    const [selectedSubtitle, setSelectedSubtitle] = useState(-1);
    const [switchingAudio, setSwitchingAudio] = useState(false);
    const [prefsMessage, setPrefsMessage] = useState("");
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
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
    // track (otherwise the user's preferred language). keepSubtitle keeps the current choice
    function fetchStream(start = 0, audio = null, keepSubtitle = undefined) {
        const params = {}
        if (start > 0) params.start = Math.floor(start)
        if (audio !== null) params.audio = audio

        return api.get(`/stream/media/${type}/${id}`, { params })
            .then(res => {
                const tracks = (res.data.subtitles ?? []).map(t => ({
                    url: `${baseURL}${t.url}`,
                    label: t.label,
                    language: t.language,
                }))
                setAudioTracks(res.data.audio ?? [])
                setSubtitles(tracks)

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
    const choicesRef = useRef({ audio: null, subtitle: null })
    useEffect(() => {
        choicesRef.current = {
            audio: audioTracks.find(a => a.selected)?.index ?? null,
            subtitle: selectedSubtitle >= 0 ? subtitles[selectedSubtitle]?.label ?? null : null,
        }
    }, [audioTracks, subtitles, selectedSubtitle])

    useEffect(() => {
        fetchStream(initialPosition)
            .then(setPlaylistUrl)
            .catch(err => setError(err.message))
            .finally(() => setLoading(false))
    }, [id])

    // set up video after playlist loads
    useEffect(() => {
        if (!playlistUrl || !videoRef.current) return;

        const video = videoRef.current;

        if (Hls.isSupported()) {
            // the playlist grows while transcoding (EVENT), so tell hls.js where to start,
            // otherwise it treats it as live and jumps to the newest segment
            const start = startAtRef.current
            const hls = new Hls({ startPosition: start > 0 ? start : 0 })
            // one remint per expiry, cleared once a fragment loads with the new token
            let reminted = false
            let resume = null
            let destroyed = false

            hls.loadSource(playlistUrl)
            hls.attachMedia(video)
            hls.on(Hls.Events.MANIFEST_PARSED, () => {
                ignoreTrackChangesRef.current = false
                setSwitchingAudio(false)
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
                const { audio, subtitle } = choicesRef.current
                fetchStream(video.currentTime, audio, subtitle)
                    .then(url => {
                        if (!destroyed) hls.loadSource(url)
                    })
                    .catch(err => setError(err.message))
            })
            return () => {
                destroyed = true
                ignoreTrackChangesRef.current = true
                hls.destroy()
                video.src = ""
            }
        }
    }, [playlistUrl])

    /**
     * Restart playback with another audio track, from the current position.
     * Each audio track is its own transcode on the server
     */
    function switchAudio(index) {
        const video = videoRef.current
        if (!video) return
        startAtRef.current = video.currentTime
        autoplayRef.current = !video.paused
        ignoreTrackChangesRef.current = true
        setSwitchingAudio(true)
        setPrefsMessage("")

        fetchStream(video.currentTime, index, currentSubtitleLabel())
            .then(setPlaylistUrl)
            .catch(err => {
                setSwitchingAudio(false)
                ignoreTrackChangesRef.current = false
                setError(err.message)
            })
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
    // is here: a <track> that fails to load never retries, even if it gets a new src later
    useEffect(() => {
        const controller = new AbortController()
        const created = []

        subtitles.forEach(t => {
            (async () => {
                for (let attempt = 0; attempt < 30 && !controller.signal.aborted; attempt++) {
                    try {
                        const res = await fetch(t.url, { signal: controller.signal })
                        if (res.ok) {
                            const blobUrl = URL.createObjectURL(new Blob([await res.text()], { type: "text/vtt" }))
                            created.push(blobUrl)
                            setSubtitleFiles(files => ({ ...files, [t.url]: blobUrl }))
                            return
                        }
                    } catch {
                        if (controller.signal.aborted) return
                    }
                    await new Promise(r => setTimeout(r, 2000))
                }
            })()
        })

        return () => {
            controller.abort()
            created.forEach(u => URL.revokeObjectURL(u))
        }
    }, [subtitles])

    // show only the selected subtitle track. <track> elements only exist for downloaded files,
    // so match them to the menu by url rather than position
    function applySubtitleSelection() {
        const video = videoRef.current
        if (!video) return
        const selectedUrl = subtitles[selectedSubtitle]?.url
        video.querySelectorAll("track").forEach(el => {
            const mode = el.dataset.url === selectedUrl ? "showing" : "disabled"
            if (el.track.mode !== mode) el.track.mode = mode
        })
    }

    useEffect(applySubtitleSelection, [subtitles, subtitleFiles, selectedSubtitle])

    // keep the menu in sync when the browser's own captions button is used. The browser also
    // turns tracks on by itself as they're added, so only a single different track (or none)
    // counts as the user's choice; anything else puts the menu's choice back
    useEffect(() => {
        const video = videoRef.current
        if (!video) return
        const textTracks = video.textTracks
        function onChange() {
            if (ignoreTrackChangesRef.current) return
            const elements = Array.from(video.querySelectorAll("track"))
            const showing = elements.filter(el => el.track.mode === "showing")
            const selectedUrl = subtitles[selectedSubtitle]?.url
            const selectedElement = elements.find(el => el.dataset.url === selectedUrl)

            if (selectedUrl && !selectedElement) {
                // the chosen track is still downloading
                applySubtitleSelection()
            } else if (showing.length === 1 && showing[0] !== selectedElement) {
                setSelectedSubtitle(subtitles.findIndex(t => t.url === showing[0].dataset.url))
            } else if (showing.length === 0 && selectedElement) {
                setSelectedSubtitle(-1)
            } else if (showing.length > 1) {
                applySubtitleSelection()
            }
        }
        textTracks.addEventListener("change", onChange)
        return () => textTracks.removeEventListener("change", onChange)
    }, [loading, subtitles, subtitleFiles, selectedSubtitle])

    // save progress every 5 seconds and on unmount
    useEffect(() => {
        function updateRefs() {
            const video = videoRef.current
            if (video) {
                positionRef.current = video.currentTime
                durationRef.current = video.duration || durationRef.current
            }
        }

        saveIntervalRef.current = setInterval(() => {
            updateRefs()
            saveProgress()
        }, 5000)

        return () => {
            clearInterval(saveIntervalRef.current)
            updateRefs()
            saveProgress()
        }
    }, [type, id])

    function saveProgress() {
        if (positionRef.current <= 0) return
        const endpoint = type === "movie" ? `/movies/${id}/progress` : `/episodes/${id}/progress`
        api.put(endpoint, {
            position_secs: positionRef.current,
            duration_secs: durationRef.current,
        }).catch(() => {})
    }

    function goToEpisode(ep) {
        navigate(`/watch/episode/${ep.id}`, {
            state: {
                position: 0,
                animeId,
                episodes: episodeList,
            }
        });
    }

    if (loading) return <Loading />
    if (error) return <Error error={error} />

    const selectedAudio = audioTracks.find(a => a.selected)

    return (
    <div className="flex flex-col h-screen bg-black">
        <div className="flex items-center justify-between px-4 py-2 bg-black/80">
            <button
                className="btn btn-ghost btn-sm text-white"
                onClick={() => animeId ? navigate(`/anime/${animeId}`) : navigate(-1)}
            >
                <ArrowLeft size={18} strokeWidth={2}/> Back
            </button>
            <div className="flex gap-2">
                <div className="dropdown dropdown-end">
                    <div tabIndex={0} role="button" className="btn btn-ghost btn-sm text-white">
                        <Languages size={18} strokeWidth={2}/> Languages
                    </div>
                    <div tabIndex={0} className="dropdown-content z-10 mt-2 w-80 rounded-box bg-base-200 p-4 shadow flex flex-col gap-3">
                        <label className="flex flex-col gap-1">
                            <span className="text-sm">Audio</span>
                            <select
                                className="select select-bordered select-sm"
                                value={selectedAudio?.index ?? ""}
                                disabled={audioTracks.length < 2 || switchingAudio}
                                onChange={e => switchAudio(Number(e.target.value))}
                            >
                                {audioTracks.length === 0 && <option value="">No audio</option>}
                                {audioTracks.map(a => (
                                    <option key={a.index} value={a.index}>{a.label}</option>
                                ))}
                            </select>
                            {switchingAudio && <span className="text-xs opacity-70">Switching audio…</span>}
                        </label>
                        <label className="flex flex-col gap-1">
                            <span className="text-sm">Subtitles</span>
                            <select
                                className="select select-bordered select-sm"
                                value={selectedSubtitle}
                                onChange={e => { setSelectedSubtitle(Number(e.target.value)); setPrefsMessage("") }}
                            >
                                <option value={-1}>Off</option>
                                {subtitles.map((t, i) => (
                                    <option key={t.url} value={i}>{t.label}</option>
                                ))}
                            </select>
                            {subtitles.length === 0 && (
                                <span className="text-xs opacity-70">This video has no text subtitles</span>
                            )}
                        </label>
                        <button className="btn btn-sm btn-primary" onClick={saveAsDefault}>
                            Use these languages by default
                        </button>
                        {prefsMessage && <span className="text-xs">{prefsMessage}</span>}
                    </div>
                </div>
                <button
                    className="btn btn-ghost btn-sm text-white"
                    disabled={!prevEpisode}
                    onClick={() => prevEpisode && goToEpisode(prevEpisode)}
                >
                    <ArrowLeft size={18} strokeWidth={2}/> Prev
                </button>
                <button
                    className="btn btn-ghost btn-sm text-white"
                    disabled={!nextEpisode}
                    onClick={() => nextEpisode && goToEpisode(nextEpisode)}
                >
                    Next <ArrowRight  size={18} strokeWidth={2}/>
                </button>
            </div>
        </div>
        <div className="flex flex-1 items-center justify-center">
            <video
                ref={videoRef}
                controls
                crossOrigin="anonymous"
                className="w-full max-w-6xl"
            >
                {subtitles.filter(t => subtitleFiles[t.url]).map(t => (
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
        </div>
    </div>
)
}
