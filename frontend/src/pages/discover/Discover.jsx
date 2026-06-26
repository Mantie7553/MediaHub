import { useEffect, useState } from "react"
import Loading from "../../components/states/Loading";
import Error from "../../components/states/Error";
import api from "../../services/api";
import ContentList from "../../components/layout/ContentList";
import ContentGrid from "../../components/layout/ContentGrid";
import useUserContent from "../../hooks/useUserContent";
import MusicDiscover from "./MusicDiscover";

const emptySections = { trending: [], popular: [], top_rated: [] };

export default function Discover() {
    const { userContentMap, refresh } = useUserContent();
    const [activeTab, setActiveTab] = useState("anime");
    const [library, setLibrary] = useState([]);
    const [query, setQuery] = useState("");
    const [sections, setSections] = useState(emptySections);
    const [searchResults, setSearchResults] = useState([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState("");

    useEffect(() => {
        if (activeTab === "music_track") return
        setSections(emptySections);
        setSearchResults([]);
        api.get(`/search?type=${activeTab}`)
            .then(resp => {
                const data = resp.data;
                setSections(data);
            })
            .catch(err => setError(err));
    }, [activeTab])

    useEffect(() => {
        if (activeTab === "music_track") return
        api.get(`/media?available=true&type=${activeTab}`)
            .then(res => setLibrary(res.data ?? []))
            .catch(() => {})
    }, [activeTab])

    function handleSearch() {
        setSections(emptySections);
        setSearchResults([]);
        if (!query.trim()) return;
        setLoading(true);
        setError("");
        api.get(`/search?type=${activeTab}&q=${query}`)
            .then(resp => setSearchResults(resp.data))
            .catch(err => setError(err.message))
            .finally(() => setLoading(false))
    }

    function handleTabChange(tab) {
        setActiveTab(tab);
        setSections(emptySections);
        setSearchResults([]);
        setQuery("");
    }

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
            <MusicDiscover userContentMap={userContentMap} onListChange={refresh}/>
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
                    <button className="btn btn-primary" onClick={handleSearch}>Search</button>
                </div>

                {loading && <Loading />}
                {error && <Error error={error} />}

                {/* Available Now */}
                <ContentList items={library} heading="Available Now" userContentMap={userContentMap} onListChange={refresh} />

                {/* Search Results */}
                {searchResults.length > 0 ? (
                    <ContentGrid items={searchResults} heading="Search Results" showActions={true} userContentMap={userContentMap} onListChange={refresh} />
                ) : (
                    <>
                        <ContentList items={sections.trending} heading="Trending" showActions={true} userContentMap={userContentMap} onListChange={refresh} />
                        <ContentList items={sections.popular} heading="Popular" showActions={true} userContentMap={userContentMap} onListChange={refresh} />
                        <ContentList items={sections.top_rated} heading={activeTab === "manga" ? "Latest Updates" : "Top Rated"} showActions={true} userContentMap={userContentMap} onListChange={refresh} />
                    </>
                )}
            </>
        )}
    </div>
}