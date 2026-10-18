import { useState, useRef } from "react"
import { Search, Clock, Plus } from "lucide-react"
import api from "../../services/api"
import useApi from "../../hooks/useApi"
import ErrorState from "../../components/states/ErrorState"
import EmptyState from "../../components/states/EmptyState"
import { ListSkeleton } from "../../components/states/Skeletons"
import Cover from "../../components/Cover"
import MusicRequestModal from "../../components/modals/MusicRequestModal"
import AlbumList from "../../components/layout/AlbumList"
import { toast } from "../../stores/useToastStore"

export default function MusicDiscover({ onListChange }) {
    const albums = useApi("/albums", { initial: [] });
    const recommended = useApi("/music/recommended", { initial: [] });
    const [query, setQuery] = useState("");
    const [searching, setSearching] = useState(false);
    // the finished search: its query, results and error. null before the first search
    const [search, setSearch] = useState(null);
    const dialogRef = useRef(null);
    const [limitVal, setLimitVal] = useState(15);
    const [selectedTrack, setSelectedTrack] = useState(null);

    function handleSearch() {
        const q = query.trim()
        if (!q) return
        setSearching(true)

        api.get(`/music/yt-search?q=${encodeURIComponent(q)}&limit=${limitVal}`)
            .then(res => setSearch({ query: q, results: res.data ?? [], error: null }))
            .catch(err => setSearch({ query: q, results: [], error: err }))
            .finally(() => setSearching(false))
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
        .then(() => { toast.success(`Downloading "${selectedTrack.title}"`); onListChange?.() })
        .catch(err => toast.error(err, "Could not request the download"))
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
            <button className="btn btn-primary" onClick={handleSearch} disabled={searching}>
                {searching ? <span className="loading loading-spinner loading-xs" /> : <Search size={16} />} Search
            </button>
            <select className="select select-bordered w-fit" value={limitVal} onChange={e => setLimitVal(Number(e.target.value))}>
                <option value={5}>5</option>
                <option value={10}>10</option>
                <option value={15}>15</option>
                <option value={20}>20</option>
                <option value={25}>25</option>
            </select>
        </div>

        {searching ? (
            <ListSkeleton rows={4} />
        ) : search?.error ? (
            <ErrorState compact error={search.error} onRetry={handleSearch} />
        ) : search && search.results.length === 0 ? (
            <EmptyState compact title={`No results for "${search.query}"`} hint="Try fewer words, or the artist's name." />
        ) : search && (
            <div className="flex flex-col gap-1">
                <h2 className="font-bold">Results for "{search.query}"</h2>
                <ul className="flex flex-col">
                    {search.results.map(track => (
                        <li key={track.id} className="flex items-center gap-3 p-3 hover:bg-base-200 rounded-lg">
                            <Cover src={track.thumbnail} alt={track.title} className="w-12 h-12 shrink-0 rounded object-cover" />
                            <div className="flex-1 min-w-0">
                                <p className="font-medium text-sm truncate">{track.title}</p>
                                <p className="text-xs opacity-70">{track.uploader}</p>
                            </div>
                            <span className="text-xs opacity-50 flex items-center gap-1">
                                <Clock size={12} />
                                {formatDuration(track.duration_secs)}
                            </span>
                            <button className="btn btn-primary btn-xs" aria-label={`Request ${track.title}`} onClick={() => { setSelectedTrack(track); dialogRef.current.showModal(); }}>
                                <Plus size={14}/>
                            </button>
                        </li>
                    ))}
                </ul>
            </div>
        )}
        <MusicRequestModal dialogRef={dialogRef} track={selectedTrack} onConfirm={handleRequest} />

        {albums.error ? (
            <div className="my-4">
                <h2 className="font-bold mb-2">Available Now</h2>
                <ErrorState compact error={albums.error} onRetry={albums.reload} />
            </div>
        ) : (
            <AlbumList albums={albums.data} heading="Available Now" loading={albums.loading}
                emptyText="No music yet" emptyHint="Search above and request tracks to download them." />
        )}
        {/* recommendations are optional, they stay hidden when empty or failed */}
        {recommended.data.length > 0 && <AlbumList albums={recommended.data} heading="Recommended" />}

    </div>
}