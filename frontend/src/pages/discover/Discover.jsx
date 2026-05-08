import { useEffect, useState } from "react"
import Loading from "../../components/states/Loading";
import api from "../../services/api";
import { SearchCard } from "../../components/cards";

/**
 * Discover page layout
 * @returns 
 */
export default function Discover() {
    const [activeTab, setActiveTab] = useState("anime");
    const [query, setQuery] = useState("");
    const [results, setResults] = useState([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState("");

    useEffect(() => {
        api.get(`/search?type=${activeTab}`)
        .then(resp => setResults(resp.data))
        .catch(err => setError(err));
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

    return <div>
        <div className="tabs tabs-lift pb-2">
            <input type="radio" name="my_tabs_3" className="tab" aria-label="Anime"
            checked={activeTab === "anime"}
            onChange={() => { setActiveTab("anime"); setResults([]); setQuery(""); }}
            />

            <input type="radio" name="my_tabs_3" className="tab" aria-label="Manga"
            checked={activeTab === "manga"}
            onChange={() => { setActiveTab("manga"); setResults([]); setQuery(""); }}
            />
        </div>
        <div className="join border rounded p-1">
            <input value={query} onChange={(e) => setQuery(e.target.value)}/>
            <button className="btn" onClick={handleSearch}>Search</button>
        </div>

        <div>
            {loading && <Loading />}
            {error && <Error error={error}/>}
            <ul className="flex flex-wrap gap-4 p-4">
                {results.map(item => <SearchCard key={item.external_id} item={item} />)}
            </ul>
        </div>

    </div>
}

