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
import CollectionList from "../../components/layout/CollectionList"
import { toast } from "../../stores/useToastStore"
import SearchField from "../../components/SearchField"
import SectionHeader from "../../components/layout/SectionHeader"
import Format from "../../utils/format"
import TrackByline, { ExplicitBadge } from "../../components/TrackByline"
import { SortMenu } from "../../components/details/Menus"
import useSort from "../../hooks/useSort"
import { sortItems } from "../../utils/sort"
import { albumSorts, artistSorts, musicResultSorts } from "../../utils/sortOptions"

export default function MusicDiscover({ onListChange }) {
    const albums = useApi("/albums", { initial: [] });
    const recommended = useApi("/music/recommended", { initial: [] });
    // a playlist per artist on the server
    const artistPlaylists = useApi("/artist-playlists", { initial: [] });
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
            .then(res => setSearch({ query: q, songs: res.data?.songs ?? [], videos: res.data?.videos ?? [], error: null }))
            .catch(err => setSearch({ query: q, songs: [], videos: [], error: err }))
            .finally(() => setSearching(false))
    }

    function openRequest(track) {
        setSelectedTrack(track)
        dialogRef.current.showModal()
    }

    function handleRequest({ artist, album, trackNumber }) {
        api.post("/search/save", {
            external_id: selectedTrack.id,
            external_source: "ytdlp",
            title: selectedTrack.title,
            cover_image_url: selectedTrack.thumbnail,
            type: "music_track",
            source_url: selectedTrack.url,
            artist,
            album,
            track_number: trackNumber,
            duration_secs: selectedTrack.duration_secs,
            action: "download",
        })
        .then(() => { toast.success(`Downloading "${selectedTrack.title}"`); onListChange?.(); artists.reload(true) })
        .catch(err => toast.error(err, "Could not request the download"))
    }

    return <div className="flex flex-col gap-10">
        {/* what the server already has comes first, ahead of anything new to request */}
        {albums.error ? (
            <section>
                <SectionHeader title="Available Now"/>
                <ErrorState compact error={albums.error} onRetry={albums.reload} />
            </section>
        ) : (
            <AlbumList albums={albums.data} heading="Available Now" loading={albums.loading} expandKey="discover:music:available" sortOptions={albumSorts}
                emptyText="No music yet" emptyHint="Search below and request tracks to download them." />
        )}
        {/* like recommendations, they stay hidden when empty or failed */}
        {artistPlaylists.data.length > 0 && <CollectionList collections={artistPlaylists.data} kind="playlist" heading="Artists" canCreate={false}
            expandKey="discover:music:artists" sortOptions={artistSorts}/>}

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
        ) : search && search.songs.length === 0 && search.videos.length === 0 ? (
            <EmptyState compact title={`No results for "${search.query}"`} hint="Try fewer words, or the artist's name." />
        ) : search && <>
            <TrackSection heading={`Songs for "${search.query}"`} tracks={search.songs} onRequest={openRequest} sortKey="discover:music:songs"/>
            <TrackSection heading={search.songs.length > 0 ? "Videos" : `Videos for "${search.query}"`} tracks={search.videos} onRequest={openRequest} sortKey="discover:music:videos"/>
        </>}
        <MusicRequestModal dialogRef={dialogRef} track={selectedTrack} artists={artists.data} onConfirm={handleRequest} />

        {/* recommendations are optional, they stay hidden when empty or failed */}
        {recommended.data.length > 0 && <AlbumList albums={recommended.data} heading="Recommended" expandKey="discover:music:recommended" sortOptions={albumSorts}/>}

    </div>
}

/**
 * A list of music search results, hidden when there are none
 * @param {string} heading the section's title
 * @param {object[]} tracks the results
 * @param {function} onRequest called with a track when its download button is pressed
 * @param {string} sortKey remembers the sort across reloads under this name (see useSort)
 */
function TrackSection({ heading, tracks: given, onRequest, sortKey }) {
    const [sort, setSort] = useSort(sortKey, musicResultSorts)
    if (given.length === 0) return null
    const tracks = sortItems(given, sort)
    return <section>
        <SectionHeader title={heading} count={tracks.length}>
            {tracks.length > 1 && <SortMenu options={musicResultSorts} value={sort} onChange={setSort} label={`Sort ${heading}`}/>}
        </SectionHeader>
        <ul className="surface flex flex-col p-1.5">
            {tracks.map(track => (
                <li key={track.id} className="flex items-center gap-3 p-2.5 hover:bg-white/5 rounded-field transition-colors">
                    <Cover src={track.thumbnail} alt={track.title} className="size-12 shrink-0 rounded-lg object-cover" />
                    <div className="flex-1 min-w-0">
                        <p className="flex items-center gap-1.5 min-w-0">
                            <span className="font-medium text-sm truncate">{track.title}</span>
                            {track.explicit && <ExplicitBadge/>}
                        </p>
                        <TrackByline track={track} className="text-xs text-base-content/60"/>
                    </div>
                    <span className="hidden sm:flex text-xs text-base-content/50 items-center gap-1 tabular-nums">
                        <Clock size={12} />
                        {track.duration_secs ? Format.duration(track.duration_secs) : "--:--"}
                    </span>
                    <button className="btn btn-primary btn-sm btn-circle" aria-label={`Request ${track.title}`} onClick={() => onRequest(track)}>
                        <Plus size={16}/>
                    </button>
                </li>
            ))}
        </ul>
    </section>
}
