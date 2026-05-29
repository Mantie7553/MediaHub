import { useEffect, useState} from "react";
import { NavLink, useParams} from "react-router-dom";
import api from "../../services/api";
import Loading from "../../components/states/Loading";
import Error from "../../components/states/Error";

export default function AnimeViewPage() {
    const [anime, setAnime] = useState(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState("");
    const [requesting, setRequesting] = useState(false);
    const [requestMsg, setRequestMsg] = useState("");
    const { id } = useParams();

    useEffect(() => {
        setLoading(true);
        api.get(`/media/${id}`)
        .then(resp => setAnime(resp.data))
        .catch(err => setError(err.message))
        .finally(() => setLoading(false))
    }, [])

    function handleRequest() {
        setRequesting(true);
        setRequestMsg("");
        api.post("/requests", { media_item_id: id })
        .then(() => setRequestMsg("Download requested!"))
        .catch(err => setRequestMsg(err.response?.data?.error ?? err.message))
        .finally(() => setRequesting(false))
    }

    if (loading) return <Loading />
    if (error) return <Error error={error} />
    if (!anime) return null

    
}