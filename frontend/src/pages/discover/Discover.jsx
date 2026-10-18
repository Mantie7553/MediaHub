import { useState } from "react"
import ErrorState from "../../components/states/ErrorState";
import { CardGridSkeleton } from "../../components/states/Skeletons";
import api from "../../services/api";
import useApi from "../../hooks/useApi";
import ContentList from "../../components/layout/ContentList";
import ContentGrid from "../../components/layout/ContentGrid";
import useUserContent from "../../hooks/useUserContent";
import MusicDiscover from "./MusicDiscover";

const emptySections = { trending: [], popular: [], top_rated: [] };

export default function Discover() {
    const { userContentMap, refresh } = useUserContent();
    const [activeTab, setActiveTab] = useState("anime");
    const [query, setQuery] = useState("");
    const isMusic = activeTab === "music_track";
    const sections = useApi(isMusic ? null : `/search?type=${activeTab}`, { initial: emptySections });
    const library = useApi(isMusic ? null : `/media?available=true&type=${activeTab}`, { initial: [] });
    // the finished search: its query, results and error. null when not searching
    const [search, setSearch] = useState(null);
    const [searching, setSearching] = useState(false);

    function handleSearch() {
        const q = query.trim();
        if (!q) {
            setSearch(null);
            return;
        }
        setSearching(true);
        api.get(`/search?type=${activeTab}&q=${encodeURIComponent(q)}`)
            .then(resp => setSearch({ query: q, results: resp.data ?? [], error: null }))
            .catch(err => setSearch({ query: q, results: [], error: err }))
            .finally(() => setSearching(false))
    }

    function handleTabChange(tab) {
        setActiveTab(tab);
        setSearch(null);
        setQuery("");
    }

    const listProps = { showActions: true, userContentMap, onListChange: refresh, loading: sections.loading };
    const topRatedHeading = activeTab === "manga" ? "Latest Updates" : "Top Rated";

    return <div className="flex flex-col gap-6">
        {/* Tabs */}
        <div className="tabs tabs-lift">
            <input type="radio" name="tabs" className="tab" aria-label="Anime"
                checked={activeTab === "anime"}
                onChange={() => handleTabChange("anime")}
            />
            <input type="radio" name="tabs" className="tab" aria-label="Manga"
                checked={activeTab === "manga"}
                onChange={() => handleTabChange("manga")}
            />
            <input type="radio" name="tabs" className="tab" aria-label="Movies"
                checked={activeTab === "movie"}
                onChange={() => handleTabChange("movie")}
            />
            <input type="radio" name="tabs" className="tab" aria-label="Music"
                checked={activeTab === "music_track"}
                onChange={() => handleTabChange("music_track")}
            />
        </div>

        {activeTab === "music_track" ? (
            <MusicDiscover onListChange={refresh}/>
        ) : (
            <>
                {/* Search */}
                <div className="flex gap-2">
                    <input
                        className="input input-bordered flex-1 max-w-1/2"
                        placeholder={`Search ${activeTab}...`}
                        value={query}
                        onChange={(e) => setQuery(e.target.value)}
                        onKeyDown={(e) => e.key === "Enter" && handleSearch()}
                    />
                    <button className="btn btn-primary" onClick={handleSearch} disabled={searching}>
                        {searching && <span className="loading loading-spinner loading-xs" />} Search
                    </button>
                    {search && <button className="btn btn-ghost" onClick={() => { setSearch(null); setQuery(""); }}>Clear</button>}
                </div>

                {/* Available Now */}
                {library.error ? (
                    <div className="my-4">
                        <h2 className="font-bold mb-2">Available Now</h2>
                        <ErrorState compact error={library.error} onRetry={library.reload} />
                    </div>
                ) : (
                    <ContentList items={library.data} heading="Available Now" userContentMap={userContentMap} onListChange={refresh}
                        loading={library.loading} emptyText="Nothing downloaded yet" />
                )}

                {/* Search Results */}
                {searching ? (
                    <div className="flex flex-col gap-2">
                        <h2 className="font-bold">Search Results</h2>
                        <CardGridSkeleton count={6} />
                    </div>
                ) : search ? (
                    search.error
                        ? <ErrorState compact error={search.error} onRetry={handleSearch} />
                        : <ContentGrid items={search.results} heading={`Results for "${search.query}"`} showActions={true} userContentMap={userContentMap} onListChange={refresh}
                            emptyText={`No results for "${search.query}"`} emptyHint="Check the spelling or try the title in another language." />
                ) : sections.error ? (
                    <ErrorState compact error={sections.error} onRetry={sections.reload} />
                ) : (
                    <>
                        <ContentList items={sections.data.trending ?? []} heading="Trending" {...listProps} />
                        <ContentList items={sections.data.popular ?? []} heading="Popular" {...listProps} />
                        <ContentList items={sections.data.top_rated ?? []} heading={topRatedHeading} {...listProps} />
                    </>
                )}
            </>
        )}
    </div>
}