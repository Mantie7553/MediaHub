import { useEffect, useState } from "react";
import ScrollPicker from "../ScrollPicker";
import Format from "../../utils/format";

export default function AddToListModal({item, onConfirm, dialogRef, initialValues}) {
    const [status, setStatus] = useState(initialValues?.status ?? 'planned');
    const [score, setScore] = useState(initialValues?.rating ?? 0);
    const [progress, setProgress] = useState((initialValues?.episodes_watched || initialValues?.chapters_read) ?? 0);

    useEffect(() => {
        setStatus(initialValues?.status ?? 'planned');
        setScore(initialValues?.rating ?? 0);
        setProgress((initialValues?.episodes_watched || initialValues?.chapters_read) ?? 0);
    }, [initialValues]);

    function statusOptions(type) {
        switch(type) {
            case "movie": return ["current", "completed", "wishlist"]
            default: return ["current", "completed", "dropped", "planned", "wishlist"]
        }
    }

    return <dialog className="modal" ref={dialogRef} onClick={e => { e.stopPropagation(); e.preventDefault(); }}>
        <div className="modal-box">
            <h2>{item.title || item.media_title}</h2>
            <section>
                <h4>Status</h4>
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
                <section>
                    <h4>Progress</h4>
                    <ScrollPicker value={progress} onChange={setProgress} max={(item.episode_count || item.total_chapters) ?? 999}/>
                </section>
            )}
            <section>
                <h4>Score</h4>
                <ScrollPicker value={score} onChange={setScore} max={10}/>
            </section>
            <section>
                <button className="btn btn-primary" onClick={() => {onConfirm({ status, score, progress}); dialogRef.current.close()}}>Confirm</button>
                <button className="btn btn-secondary" onClick={() => dialogRef.current.close()}>Cancel</button>
            </section>
        </div>
    </dialog>
}