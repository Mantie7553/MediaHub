import { useEffect, useState } from "react"
import api from "../../services/api"
import { MangaCard } from "../../components/cards"

export default function LibraryPage() {
    const [content, setContent] = useState([]);
    const [error, setError] = useState("");
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        setLoading(true);
        api.get("/media?type=manga")
        .then(resp => setContent(resp.data))
        .catch(err => setError(err.message))
        .finally(() => setLoading(false))
    }, []);

    if (loading) return <div className="flex justify-center p-10"><span className="loading loading-spinner loading-lg"></span></div>

    if (error) return <div className="alert alert-error">{error}</div>

    return <ul className="flex flex-wrap gap-4 p-4">
        {content.map(item => <MangaCard key={item.id} item={item} />)}
    </ul>
}