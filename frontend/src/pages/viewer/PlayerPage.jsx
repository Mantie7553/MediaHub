import { useState, useEffect, useRef } from "react"
import { useParams, useLocation, useNavigate } from "react-router-dom"
import Hls from "hls.js"
import api from "../../services/api"
import Loading from "../../components/states/Loading"
import Error from "../../components/states/Error"
import { ArrowLeft, ArrowRight } from "lucide-react"

export default function PlayerPage() {
    const location = useLocation();
    const navigate = useNavigate();
    const { type, id } = useParams();
    const videoRef = useRef(null);
    const [playlistUrl, setPlaylistUrl] = useState(null);
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

    const baseURL = import.meta.env.VITE_API_URL

    useEffect(() => {
        api.get(`/stream/media/${type}/${id}`)
            .then(res => setPlaylistUrl(`${baseURL}${res.data.playlist}`))
            .catch(err => setError(err.message))
            .finally(() => setLoading(false))
    }, [id])

    // set up video after playlist loads
    useEffect(() => {
        if (!playlistUrl || !videoRef.current) return;

        const video = videoRef.current;

        if (Hls.isSupported()) {
            const hls = new Hls()
            hls.loadSource(playlistUrl)
            hls.attachMedia(video)
            hls.on(Hls.Events.MANIFEST_PARSED, () => {
                video.play().catch(() => {})
                if (initialPosition > 0) {
                    const seekOnce = () => {
                        if (video.duration && initialPosition < video.duration - 10) {
                            video.currentTime = initialPosition
                            video.removeEventListener("canplay", seekOnce)
                        }
                    }
                    video.addEventListener("canplay", seekOnce)
                }
            })
            return () => {
                hls.destroy()
                video.src = ""
            }
        }
    }, [playlistUrl, initialPosition])

    // save progress every 5 seconds and on unmount
    useEffect(() => {
        if (type !== "episode") return

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
        if (type !== "episode") return
        if (positionRef.current <= 0) return
        api.put(`/episodes/${id}/progress`, {
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
                {playlistUrl && (
                    <track
                        kind="subtitles"
                        src={`${baseURL}/stream/segments/${type}/${id}/subs.vtt`}
                        srcLang="en"
                        label="English"
                        default
                    />
                )}
            </video>
        </div>
    </div>
)
}