import { useEffect, useState } from "react";
import api from "../services/api";

export default function useUserContent() {
    const [userContent, setUserContent] = useState([]);
    const [error, setError] = useState("");

    function refresh() {
        api.get("/me/media")
            .then(resp => setUserContent(resp.data))
            .catch(err => setError(err.message ?? "Unable to retrieve user tracked media"));
    }

    useEffect(() => {
        refresh();
    }, [])

    const userContentMap = Object.fromEntries([
        ...userContent.map(item => [item.media_item_id, item]),
        ...userContent.filter(item => item.external_id != null).map(item => [String(item.external_id), item]),
        ...userContent.filter(item => item.album_id != null).map(item => [item.album_id, item]),
    ]);

    return { userContent, userContentMap, error, refresh };
}