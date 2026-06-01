import { useState, useEffect, useRef } from "react"
import { useParams } from "react-router-dom"
import Hls from "hls.js"
import api from "../../services/api"
import Loading from "../../components/states/Loading"
import Error from "../../components/states/Error"

export default function PlayerPage() {
    const { type, id } = useParams()
    const videoRef = useRef(null)
    const [playlistUrl, setPlaylistUrl] = useState(null)
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState(null)

    const baseURL = import.meta.env.VITE_API_URL

    useEffect(() => {
        api.get(`/stream/media/${type}/${id}`)
            .then(res => setPlaylistUrl(`${baseURL}${res.data.playlist}`))
            .catch(err => setError(err.message))
            .finally(() => setLoading(false))
    }, [id])

    useEffect(() => {
        if (!playlistUrl || !videoRef.current) return

        if (Hls.isSupported()) {
            const hls = new Hls()
            hls.loadSource(playlistUrl)
            hls.attachMedia(videoRef.current)
            return () => hls.destroy()
        } else if (videoRef.current.canPlayType("application/vnd.apple.mpegurl")) {
            videoRef.current.src = playlistUrl
        }
    }, [playlistUrl])

    if (loading) return <Loading />
    if (error) return <Error error={error} />

    return (
        <div className="flex items-center justify-center w-full h-full bg-black">
            <video
                ref={videoRef}
                controls
                autoPlay
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
    )
}