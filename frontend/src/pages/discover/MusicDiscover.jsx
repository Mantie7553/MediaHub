import { useState, useRef } from "react"
import { Search, Clock, Music, Plus } from "lucide-react"
import api from "../../services/api"
import Loading from "../../components/states/Loading"
import Error from "../../components/states/Error"
import MusicRequestModal from "../../components/modals/MusicRequestModal"
import useAudioStore from "../../stores/useAudioStore"

export default function MusicDiscover({ userContentMap, onListChange }) {
    const [query, setQuery] = useState("")
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState("")
    const [ytResults, setYtResults] = useState([])
    const dialogRef = useRef(null)
    const [selectedTrack, setSelectedTrack] = useState(null)
    const play = useAudioStore(state => state.play)

    function handleSearch() {
        if (!query.trim()) return
        setLoading(true)
        setError("")

        api.get(`/music/yt-search?q=${encodeURIComponent(query)}`)
            .then(res => setYtResults(res.data ?? []))
            .catch(err => setError(err.message))
            .finally(() => setLoading(false))
    }

    function formatDuration(secs) {
        const mins = Math.floor(secs / 60)
        const s = secs % 60
        return `${mins}:${String(s).padStart(2, "0")}`
    }

    function handleRequest({ artist, album }) {
        api.post("/search/save", {
            external_id: selectedTrack.id,
            external_source: "ytdlp",
            title: selectedTrack.title,
            cover_image_url: selectedTrack.thumbnail,
            type: "music_track",
            source_url: selectedTrack.url,
            artist,
            album,
            duration_secs: selectedTrack.duration_secs,
            action: "download",
        })
        .then(() => onListChange?.())
        .catch(err => setError(err.response?.data ?? err.message))
    }

    return <div className="flex flex-col gap-4">
        {/* Search Bar */}
        <div className="flex gap-2">
            <input
                className="input input-bordered flex-1 max-w-1/2"
                placeholder="Search YouTube Music..."
                value={query}
                onChange={e => setQuery(e.target.value)}
                onKeyDown={e => e.key === "Enter" && handleSearch()}
            />
            <button className="btn btn-primary" onClick={handleSearch}>
                <Search size={16} /> Search
            </button>
        </div>

        {loading && <Loading />}
        {error && <Error error={error} />}

        {ytResults.length > 0 && (
            <div className="flex flex-col gap-1">
                <h2 className="font-bold">Results</h2>
                <ul className="flex flex-col">
                    {ytResults.map(track => (
                        <li key={track.id} className="flex items-center gap-3 p-3 hover:bg-base-200 rounded-lg">
                            {track.thumbnail ? (
                                <img src={track.thumbnail} className="w-12 h-12 rounded object-cover" />
                            ) : (
                                <div className="w-12 h-12 rounded bg-base-300 flex items-center justify-center">
                                    <Music size={20} />
                                </div>
                            )}
                            <div className="flex-1 min-w-0">
                                <p className="font-medium text-sm truncate">{track.title}</p>
                                <p className="text-xs opacity-70">{track.uploader}</p>
                            </div>
                            <span className="text-xs opacity-50 flex items-center gap-1">
                                <Clock size={12} />
                                {formatDuration(track.duration_secs)}
                            </span>
                            <button className="btn btn-primary btn-xs" onClick={() => { setSelectedTrack(track); dialogRef.current.showModal(); }}>
                                <Plus size={14}/>
                            </button>
                        </li>
                    ))}
                </ul>
            </div>
        )}
        <MusicRequestModal dialogRef={dialogRef} track={selectedTrack} onConfirm={handleRequest} />
    </div>
}