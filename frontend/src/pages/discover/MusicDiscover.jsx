import { useState, useRef } from "react"
import { Clock, Plus } from "lucide-react"
import api from "../../services/api"
import useApi from "../../hooks/useApi"
import ErrorState from "../../components/states/ErrorState"
import EmptyState from "../../components/states/EmptyState"
import { ListSkeleton } from "../../components/states/Skeletons"
import Cover from "../../components/Cover"
import MusicRequestModal from "../../components/modals/MusicRequestModal"
import AlbumList from "../../components/layout/AlbumList"
import { toast } from "../../stores/useToastStore"
import SearchField from "../../components/SearchField"
import SectionHeader from "../../components/layout/SectionHeader"
import Format from "../../utils/format"

export default function MusicDiscover({ onListChange }) {
    const albums = useApi("/albums", { initial: [] });
    const recommended = useApi("/music/recommended", { initial: [] });
    // suggestions for the request form; it still works without them
    const artists = useApi("/music/artists", { initial: [] });
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
        .then(() => { toast.success(`Downloading "${selectedTrack.title}"`); onListChange?.(); artists.reload(true) })
        .catch(err => toast.error(err, "Could not request the download"))
    }

    return <div className="flex flex-col gap-10">
        <SearchField
            value={query}
            onChange={setQuery}
            onSubmit={handleSearch}
            onClear={() => { setSearch(null); setQuery(""); }}
            searching={searching}
            placeholder="Search YouTube Music…"
        >
            <select className="select select-ghost select-sm w-fit bg-white/5 rounded-lg" value={limitVal} onChange={e => setLimitVal(Number(e.target.value))} aria-label="Number of results">
                <option value={5}>5</option>
                <option value={10}>10</option>
                <option value={15}>15</option>
                <option value={20}>20</option>
                <option value={25}>25</option>
            </select>
        </SearchField>

        {searching ? (
            <ListSkeleton rows={4} />
        ) : search?.error ? (
            <ErrorState compact error={search.error} onRetry={handleSearch} />
        ) : search && search.results.length === 0 ? (
            <EmptyState compact title={`No results for "${search.query}"`} hint="Try fewer words, or the artist's name." />
        ) : search && (
            <section>
                <SectionHeader title={`Results for "${search.query}"`} count={search.results.length}/>
                <ul className="surface flex flex-col p-1.5">
                    {search.results.map(track => (
                        <li key={track.id} className="flex items-center gap-3 p-2.5 hover:bg-white/5 rounded-field transition-colors">
                            <Cover src={track.thumbnail} alt={track.title} className="size-12 shrink-0 rounded-lg object-cover" />
                            <div className="flex-1 min-w-0">
                                <p className="font-medium text-sm truncate">{track.title}</p>
                                <p className="text-xs text-base-content/60 truncate">{track.uploader}</p>
                            </div>
                            <span className="hidden sm:flex text-xs text-base-content/50 items-center gap-1 tabular-nums">
                                <Clock size={12} />
                                {track.duration_secs ? Format.duration(track.duration_secs) : "--:--"}
                            </span>
                            <button className="btn btn-primary btn-sm btn-circle" aria-label={`Request ${track.title}`} onClick={() => { setSelectedTrack(track); dialogRef.current.showModal(); }}>
                                <Plus size={16}/>
                            </button>
                        </li>
                    ))}
                </ul>
            </section>
        )}
        <MusicRequestModal dialogRef={dialogRef} track={selectedTrack} artists={artists.data} onConfirm={handleRequest} />

        {albums.error ? (
            <section>
                <SectionHeader title="Available Now"/>
                <ErrorState compact error={albums.error} onRetry={albums.reload} />
            </section>
        ) : (
            <AlbumList albums={albums.data} heading="Available Now" loading={albums.loading}
                emptyText="No music yet" emptyHint="Search above and request tracks to download them." />
        )}
        {/* recommendations are optional, they stay hidden when empty or failed */}
        {recommended.data.length > 0 && <AlbumList albums={recommended.data} heading="Recommended" />}

    </div>
}
