import { useEffect, useState } from "react"
import api from "../../services/api"
import { MangaCard } from "../../components/cards"

/**
 * Manga Library page layout
 * @returns
 */
export default function LibraryPage() {
    const [content, setContent] = useState([]);
    const [error, setError] = useState("");
    const [loading, setLoading] = useState(false);
    const [title, setTitle] = useState("");
    const [adding, setAdding] = useState(false);

    function fetchManga() {
        setLoading(true);
        api.get("/media?type=manga")
        .then(resp => setContent(resp.data))
        .catch(err => setError(err.message))
        .finally(() => setLoading(false))
    }

    useEffect(() => { fetchManga() }, []);

    function handleAdd() {
        if (!title.trim()) return;
        setAdding(true);
        api.post("/media", { type: "manga", title: title.trim() })
        .then(() => { setTitle(""); document.getElementById("add_manga_modal").close(); fetchManga(); })
        .catch(err => setError(err.message))
        .finally(() => setAdding(false))
    }

    if (loading) return <div className="flex justify-center p-10"><span className="loading loading-spinner loading-lg"></span></div>
    if (error) return <div className="alert alert-error">{error}</div>

    return <>
        <div className="flex justify-end p-4">
            <button className="btn btn-primary" onClick={() => document.getElementById("add_manga_modal").showModal()}>+ Add Manga</button>
        </div>

        <ul className="flex flex-wrap gap-4 p-4">
            {content.map(item => <MangaCard key={item.id} item={item} />)}
        </ul>

        <dialog id="add_manga_modal" className="modal">
            <div className="modal-box">
                <h3 className="font-bold text-lg mb-4">Add Manga</h3>
                <input
                    type="text"
                    placeholder="Title"
                    className="input input-bordered w-full"
                    value={title}
                    onChange={e => setTitle(e.target.value)}
                    onKeyDown={e => e.key === "Enter" && handleAdd()}
                />
                <div className="modal-action">
                    <button className="btn" onClick={() => document.getElementById("add_manga_modal").close()}>Cancel</button>
                    <button className="btn btn-primary" onClick={handleAdd} disabled={adding}>
                        {adding ? <span className="loading loading-spinner loading-sm"></span> : "Add"}
                    </button>
                </div>
            </div>
            <form method="dialog" className="modal-backdrop"><button>close</button></form>
        </dialog>
    </>
}