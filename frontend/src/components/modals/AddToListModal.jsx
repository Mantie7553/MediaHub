import { useEffect, useState } from "react";
import ScrollPicker from "../ScrollPicker";
import Format from "../../utils/format";

export default function AddToListModal({item, onConfirm, dialogRef, initialValues, isLibraryItem}) {
    const [status, setStatus] = useState(initialValues?.status ?? 'planned');
    const [score, setScore] = useState(initialValues?.rating ?? 0);
    const [progress, setProgress] = useState((initialValues?.episodes_watched || initialValues?.chapters_read) ?? 0);

    useEffect(() => {
        setStatus(initialValues?.status ?? 'planned');
        setScore(initialValues?.rating ?? 0);
        setProgress((initialValues?.episodes_watched || initialValues?.chapters_read) ?? 0);
    }, [initialValues]);

    function statusOptions(type) {
        const withWishlist = !isLibraryItem;
        switch(type) {
            case "music_album": return ["completed", "planned"]
            case "movie": return ["completed", "current", ...(withWishlist ? ["wishlist"] : [])]
            default: return ["completed", "current", "dropped", "planned", ...(withWishlist ? ["wishlist"] : [])]
        }
    }

    return <dialog className="modal" ref={dialogRef} onClick={e => { e.stopPropagation(); e.preventDefault(); }}>
        <div className="modal-box flex flex-col gap-4">
            <h2 className="text-xl font-semibold">{item.title || item.media_title}</h2>
            <section>
                <h4 className="text-xs font-medium text-base-content/80 uppercase tracking-wide mb-1">Status</h4>
                <div className="flex flex-wrap gap-2">
                    {statusOptions(item.type || item.media_type).map(option => (
                        <button
                            key={option}
                            type="button"
                            className={`btn btn-sm ${option === status ? "btn-primary" : "btn-outline"}`}
                            onClick={() => setStatus(option)}
                        >
                            {Format.statusLabel(option, item.type ?? item.media_type)}
                        </button>
                    ))}
                </div>
            </section>
            {((item.type || item.media_type) !== "movie" && (item.type || item.media_type) !== "anime") && (
                <section className="pl-4">
                    <h4 className="text-xs font-medium text-base-content/80 uppercase tracking-wide mb-1">Progress</h4>
                    <ScrollPicker value={progress} onChange={setProgress} max={item.total_chapters ?? 999}/>
                </section>
            )}
            <section className="pl-4">
                <h4 className="text-xs font-medium text-base-content/80 uppercase tracking-wide mb-1">Score</h4>
                <ScrollPicker value={score} onChange={setScore} max={10}/>
            </section>
            <section className="flex gap-2 pt-2 pl-4">
                <button className="btn btn-primary" onClick={() => {onConfirm({ status, score, progress}); dialogRef.current.close()}}>Confirm</button>
                <button className="btn btn-secondary btn-outline" onClick={() => dialogRef.current.close()}>Cancel</button>
            </section>
        </div>
    </dialog>
}