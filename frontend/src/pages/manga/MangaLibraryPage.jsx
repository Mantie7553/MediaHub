import { useEffect, useState } from "react"
import api from "../../services/api"
import { MangaCard } from "../../components/cards"
import Error from "../../components/states/Error"
import Loading from "../../components/states/Loading"

/**
 * Manga Library page layout
 * @returns
 */
export default function LibraryPage() {
    const [content, setContent] = useState([]);
    const [error, setError] = useState("");
    const [loading, setLoading] = useState(false);

    function fetchManga() {
        setLoading(true);
        api.get("/media?type=manga")
        .then(resp => setContent(resp.data))
        .catch(err => setError(err.message))
        .finally(() => setLoading(false))
    }

    useEffect(() => { fetchManga() }, []);

    if (loading) return <Loading />
    if (error) return <Error error={error} />

    return <ul className="flex flex-wrap gap-4 p-4">
        {content.map(item => <MangaCard key={item.id} item={item} />)}
    </ul>
}