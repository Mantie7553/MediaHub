import { useState } from "react";
import api from "../../services/api";

/**
 * Display a search result in a card with options to add to list or request download
 * @param {any} item the search result this displays
 * @returns
 */
export default function SearchCard({ item }) {
    const [msg, setMsg] = useState("");
    const [loading, setLoading] = useState(false);

    function handleAction(action) {
        setLoading(true);
        setMsg("");
        api.post("/search/save", {
            external_id: item.external_id,
            external_source: item.external_source,
            title: item.title,
            cover_image_url: item.cover_image_url,
            type: item.type,
            action: action,
        })
        .then(() => setMsg(action === "list" ? "Added to list!" : "Download requested!"))
        .catch(err => setMsg(err.response?.data?.error ?? err.message))
        .finally(() => setLoading(false))
    }

    return <li className="card border border-base-300 w-48 shrink-0">
        <figure>
            {item.cover_image_url ? (
                <img src={item.cover_image_url} className="w-full h-48" />
            ) : (
                <div className="skeleton h-48 w-full"></div>
            )}
        </figure>
        <div className="card-body p-3 gap-2">
            <h3 className="card-title text-sm">{item.title}</h3>
            <div className="flex flex-col gap-1">
                <button className="btn btn-sm btn-primary" onClick={() => handleAction("list")} disabled={loading}>
                    + Add to List
                </button>
                <button className="btn btn-sm btn-outline" onClick={() => handleAction("download")} disabled={loading}>
                    Request Download
                </button>
            </div>
            {msg && <p className="text-xs mt-1">{msg}</p>}
        </div>
    </li>
}