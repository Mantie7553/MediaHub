import { useEffect, useState } from "react"
import Loading from "../../components/states/Loading";
import api from "../../services/api";
import { Card } from "../../components/cards";
import ContentList from "../../components/layout/ContentList";
import ContentGrid from "../../components/layout/ContentGrid";

/**
 * Discover page layout
 * @returns 
 */
export default function Discover() {
    const [activeTab, setActiveTab] = useState("anime");
    const [library, setLibrary] = useState([]);
    const [query, setQuery] = useState("");
    const [results, setResults] = useState([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState("");

    useEffect(() => {
        api.get(`/search?type=${activeTab}`)
        .then(resp => setResults(resp.data))
        .catch(err => setError(err));
    }, [activeTab])

    useEffect(() => {
    api.get(`/media?available=true&type=${activeTab === "anime" ? "anime" : "movie"}`)
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
            <input type="radio" name="my_tabs_3" className="tab" aria-label="Anime"
            checked={activeTab === "anime"}
            onChange={() => { setActiveTab("anime"); setResults([]); setQuery(""); }}
            />
            <input type="radio" name="my_tabs_3" className="tab" aria-label="Manga"
            checked={activeTab === "manga"}
            onChange={() => { setActiveTab("manga"); setResults([]); setQuery(""); }}
            />
        </div>

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
        <ContentList items={library} heading="Available Now" />

        {/* Search Results */}
        <div>
            {loading && <Loading />}
            {error && <Error error={error}/>}
            <ContentGrid items={results} heading="Trending Now" showActions={true} />
        </div>
    </div>
}

