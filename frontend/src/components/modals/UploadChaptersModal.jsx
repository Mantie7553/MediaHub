import { useState } from "react";
import { AlertCircle, Check, FileArchive, Upload, X } from "lucide-react";
import { uploadChapter } from "../../services/mediaService";
import { errorMessage } from "../../utils/errors";
import { chapterNumberFromName, isChapterArchive } from "../../utils/chapters";
import { toast } from "../../stores/useToastStore";

let nextKey = 1;

// rows without a number yet sort last
const sortValue = number => number === "" || isNaN(Number(number)) ? Infinity : Number(number);

/**
 * Admin: add chapter files to a manga. Each file's chapter number is read from its name and
 * can be corrected before uploading. Files go up one at a time, so each stays under the
 * proxy's upload limit and a failure only affects that chapter
 * @param {object} dialogRef ref for the <dialog>
 * @param {object} manga { id, title }
 * @param {number[]} haveNumbers chapter numbers already in the library
 * @param {function} onUploaded called after the uploads finish, to reload the chapter list
 */
export default function UploadChaptersModal({ dialogRef, manga, haveNumbers = [], onUploaded }) {
    // { key, file, number, status: ready | uploading | done | error, progress, error }
    const [rows, setRows] = useState([]);
    const [uploading, setUploading] = useState(false);
    const [dragging, setDragging] = useState(false);

    const have = new Set(haveNumbers.map(Number));

    function addFiles(fileList) {
        const files = [...fileList];
        const accepted = files.filter(f => isChapterArchive(f.name));
        if (accepted.length < files.length) toast.error("Only .cbz and .zip files can be added as chapters");
        setRows(prev => [...prev, ...accepted.map(file => ({
            key: nextKey++,
            file,
            number: chapterNumberFromName(file.name, manga.title) ?? "",
            status: "ready",
            progress: 0,
            error: null,
        }))].sort((a, b) => sortValue(a.number) - sortValue(b.number)));
    }

    function update(key, changes) {
        setRows(prev => prev.map(r => r.key === key ? { ...r, ...changes } : r));
    }

    // why a ready row can't be uploaded, or null when it can
    function problem(row, all) {
        const n = row.number.trim();
        if (n === "" || isNaN(Number(n)) || Number(n) < 0) return "Enter the chapter number";
        if (have.has(Number(n))) return "Already in the library";
        if (all.some(r => r !== row && r.status !== "done" && Number(r.number) === Number(n))) return "Two files have this number";
        return null;
    }

    const ready = rows.filter(r => r.status === "ready" || r.status === "error");
    const blocked = ready.some(r => problem(r, rows));

    async function handleUpload() {
        setUploading(true);
        let added = 0;
        for (const row of ready) {
            update(row.key, { status: "uploading", progress: 0, error: null });
            try {
                await uploadChapter(manga.id, row.file, row.number.trim(), p => update(row.key, { progress: p }));
                update(row.key, { status: "done", progress: 1 });
                added++;
            } catch (err) {
                update(row.key, { status: "error", error: errorMessage(err, "Upload failed") });
            }
        }
        setUploading(false);
        if (added > 0) {
            toast.success(`Added ${added} ${added === 1 ? "chapter" : "chapters"}`);
            onUploaded?.();
        }
    }

    function reset() {
        setRows([]);
        setDragging(false);
    }

    const done = rows.length > 0 && rows.every(r => r.status === "done");

    return <dialog className="modal modal-bottom sm:modal-middle" ref={dialogRef} onClose={reset}>
        <div className="modal-box flex flex-col gap-4 sm:max-w-2xl">
            <div className="flex items-center gap-3">
                <span className="grid size-10 place-items-center rounded-full bg-primary/15 text-primary"><Upload size={18}/></span>
                <div className="min-w-0">
                    <h3 className="text-xl font-semibold">Add chapters</h3>
                    <p className="text-sm text-base-content/50 truncate">{manga.title}</p>
                </div>
            </div>

            <label
                className={`flex flex-col items-center gap-2 rounded-box border-2 border-dashed px-4 py-6 text-center cursor-pointer transition-colors
                    ${dragging ? "border-primary bg-primary/8" : "border-white/15 hover:border-white/30 hover:bg-white/[0.03]"}
                    ${uploading ? "pointer-events-none opacity-50" : ""}`}
                onDragOver={e => { e.preventDefault(); setDragging(true); }}
                onDragLeave={() => setDragging(false)}
                onDrop={e => { e.preventDefault(); setDragging(false); addFiles(e.dataTransfer.files); }}
            >
                <FileArchive size={26} className="text-base-content/40"/>
                <span className="text-sm font-medium">Drop .cbz or .zip files here, or tap to choose</span>
                <span className="text-xs text-base-content/45">One file per chapter. The number is read from each file's name, check it below.</span>
                <input type="file" multiple accept=".cbz,.zip" className="hidden"
                    onChange={e => { addFiles(e.target.files); e.target.value = ""; }}/>
            </label>

            {rows.length > 0 && <ul className="flex flex-col gap-1.5 max-h-[45dvh] overflow-y-auto -mx-1 px-1">
                {rows.map(row => {
                    const issue = row.status === "ready" || row.status === "error" ? problem(row, rows) : null;
                    return <li key={row.key} className="rounded-field bg-white/[0.03] ring-1 ring-white/5 px-3 py-2.5">
                        <div className="flex items-center gap-3">
                            <label className="flex items-center gap-1.5 shrink-0">
                                <span className="text-xs text-base-content/45">Ch.</span>
                                <input
                                    className={`input input-sm w-20 bg-base-100/60 text-base sm:text-sm tabular-nums ${issue ? "border-error/60" : "border-white/10"}`}
                                    value={row.number}
                                    inputMode="decimal"
                                    disabled={row.status === "uploading" || row.status === "done"}
                                    onChange={e => update(row.key, { number: e.target.value, status: row.status === "error" ? "ready" : row.status })}
                                    aria-label={`Chapter number for ${row.file.name}`}
                                />
                            </label>
                            <div className="flex-1 min-w-0">
                                <p className="text-sm truncate" title={row.file.name}>{row.file.name}</p>
                                <p className={`text-xs truncate ${issue || row.error ? "text-error" : "text-base-content/45"}`}>
                                    {row.error ?? issue ?? `${(row.file.size / 1048576).toFixed(1)} MB`}
                                </p>
                            </div>
                            {row.status === "done" ? <span className="grid size-7 place-items-center rounded-full bg-success/15 text-success"><Check size={15}/></span>
                                : row.status === "uploading" ? <span className="text-xs tabular-nums text-base-content/60 w-9 text-right">{Math.round(row.progress * 100)}%</span>
                                : row.status === "error" ? <AlertCircle size={18} className="text-error shrink-0"/>
                                : <button className="btn btn-ghost btn-xs btn-square" disabled={uploading}
                                    onClick={() => setRows(prev => prev.filter(r => r.key !== row.key))} aria-label={`Remove ${row.file.name}`}>
                                    <X size={14}/>
                                </button>}
                        </div>
                        {row.status === "uploading" && <div className="mt-2 h-1 rounded-full bg-white/10 overflow-hidden">
                            <div className="h-full bg-primary transition-[width] duration-200" style={{ width: `${Math.round(row.progress * 100)}%` }}></div>
                        </div>}
                    </li>
                })}
            </ul>}

            <div className="flex flex-wrap items-center justify-end gap-2 pt-1">
                {ready.length > 0 && !uploading && <span className="mr-auto text-sm text-base-content/50">
                    {ready.length} {ready.length === 1 ? "chapter" : "chapters"} to upload
                </span>}
                <button className="btn btn-ghost" onClick={() => dialogRef.current.close()} disabled={uploading}>
                    {done ? "Close" : "Cancel"}
                </button>
                {!done && <button className="btn btn-primary" onClick={handleUpload} disabled={uploading || ready.length === 0 || blocked}>
                    {uploading ? <span className="loading loading-spinner loading-xs"/> : <Upload size={16}/>}
                    {uploading ? "Uploading…" : "Upload"}
                </button>}
            </div>
        </div>
    </dialog>
}
