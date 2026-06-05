import { useEffect, useState } from "react"
import Loading from "../../components/states/Loading";
import api from "../../services/api";
import ContentList from "../../components/layout/ContentList";
import ContentGrid from "../../components/layout/ContentGrid";
import useUserContent from "../../hooks/useUserContent";
import MusicDiscover from "./MusicDiscover";

/**
 * Discover page layout
 * @returns 
 */
export default function Discover() {
    const { userContentMap, refresh } = useUserContent();
    const [activeTab, setActiveTab] = useState("anime");
    const [library, setLibrary] = useState([]);
    const [query, setQuery] = useState("");
    const [results, setResults] = useState([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState("");

    useEffect(() => {
        if (activeTab === "music_track") return
        api.get(`/search?type=${activeTab}`)
        .then(resp => setResults(resp.data))
        .catch(err => setError(err));
    }, [activeTab])

    useEffect(() => {
        if (activeTab === "music_track") return
        api.get(`/media?available=true&type=${activeTab}`)
            .then(res => setLibrary(res.data ?? []))
            .catch(() => {})
    }, [activeTab])

    /**
     * Make an API request to search for some content
     * @returns 
     */
    function handleSearch() {
        setResults([]);
        if (!query.trim()) return;
        setLoading(true);
        setError("");
        api.get(`/search?type=${activeTab}&q=${query}`)
            .then(resp => setResults(resp.data))
            .catch(err => setError(err.message))
            .finally(() => setLoading(false))
    }

return <div className="flex flex-col gap-6">
        {/* Tabs */}
        <div className="tabs tabs-lift">
            <input type="radio" name="tabs" className="tab" aria-label="Anime"
            checked={activeTab === "anime"}
            onChange={() => { setActiveTab("anime"); setResults([]); setQuery(""); }}
            />
            <input type="radio" name="tabs" className="tab" aria-label="Manga"
            checked={activeTab === "manga"}
            onChange={() => { setActiveTab("manga"); setResults([]); setQuery(""); }}
            />
            <input type="radio" name="tabs" className="tab" aria-label="Movies"
            checked={activeTab === "movie"}
            onChange={() => { setActiveTab("movie"); setResults([]); setQuery(""); }}
            />
            <input type="radio" name="tabs" className="tab" aria-label="Music"
            checked={activeTab === "music_track"}
            onChange={() => { setActiveTab("music_track"); setResults([]); setQuery(""); }}
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

                {/* Available Now */}
                <ContentList items={library} heading="Available Now" userContentMap={userContentMap} onListChange={refresh}/>

                {/* Search Results */}
                <div>
                    {loading && <Loading />}
                    {error && <Error error={error}/>}
                    <ContentGrid items={results} heading="Trending Now" showActions={true}  userContentMap={userContentMap} onListChange={refresh}/>
                </div>
            </>
        )}
    </div>
}

