import { useRef, useState } from "react";
import { createPortal } from "react-dom";
import { AlertCircle, Check, FileUp, Upload, X } from "lucide-react";
import { addEpisode, addMovieFile, addTrack, addVolume, cancelUpload, uploadInChunks } from "../../services/curationService";
import { episodeFromName, trackFromName, volumeFromName } from "../../utils/curation";
import { errorMessage } from "../../utils/errors";
import { toast } from "../../stores/useToastStore";

let nextKey = 1;

const isWhole = v => /^\d+$/.test(String(v).trim());
const ext = name => (name.match(/\.[^.]+$/)?.[0] ?? "").toLowerCase();

/*
What each kind of upload needs. fields are the details typed per file, read from the file's
name first; key(row) is what can't repeat within a title
*/
const kinds = {
    episodes: {
        uploadKind: "video",
        accept: [".mkv", ".mp4", ".m4v", ".webm", ".avi", ".mov", ".ts"],
        noun: ["episode", "episodes"],
        hint: "One file per episode. The season and episode are read from each file's name, check them below.",
        fields: [
            { key: "season", label: "Season", short: "S", width: "w-14", required: true },
            { key: "episode", label: "Episode", short: "E", width: "w-16", required: true },
            { key: "title", label: "Title (optional)", grow: true },
        ],
        fromName: (name, ctx) => {
            const { season, episode } = episodeFromName(name);
            return { season: season || ctx.defaultSeason || "1", episode, title: "" };
        },
        key: row => `${Number(row.season)}x${Number(row.episode)}`,
        taken: "That episode already has a file",
        add: (id, uploadId, row) => addEpisode(id, uploadId, {
            season_number: Number(row.season), episode_number: Number(row.episode), title: row.title.trim(),
        }),
    },
    volumes: {
        uploadKind: "epub",
        accept: [".epub"],
        noun: ["volume", "volumes"],
        hint: "One EPUB per volume. The number is read from each file's name, check it below.",
        fields: [
            { key: "number", label: "Volume", short: "Vol.", width: "w-16", required: true },
            { key: "title", label: "Title (optional)", grow: true },
        ],
        fromName: name => ({ number: volumeFromName(name), title: "" }),
        key: row => String(Number(row.number)),
        taken: "Already in the library",
        add: (id, uploadId, row) => addVolume(id, uploadId, { volume_number: Number(row.number), title: row.title.trim() }),
    },
    tracks: {
        uploadKind: "audio",
        accept: [".mp3"],
        noun: ["track", "tracks"],
        hint: "MP3 files. The track number and title are read from each file's name, check them below.",
        fields: [
            { key: "trackNumber", label: "Track", short: "#", width: "w-14" },
            { key: "title", label: "Title", grow: true, required: true, text: true },
        ],
        fromName: name => trackFromName(name),
        key: row => row.title.trim().toLowerCase(),
        taken: "The album already has a track with this title",
        add: (id, uploadId, row) => addTrack(id, uploadId, {
            title: row.title.trim(), track_number: row.trackNumber.trim() ? Number(row.trackNumber) : null,
        }),
    },
    movie: {
        uploadKind: "video",
        accept: [".mkv", ".mp4", ".m4v", ".webm", ".avi", ".mov", ".ts"],
        noun: ["movie file", "movie files"],
        single: true,
        hint: "The movie's video file. Large files are sent in pieces, keep this open until it finishes.",
        fields: [],
        fromName: () => ({}),
        key: () => "movie",
        taken: "",
        add: (id, uploadId) => addMovieFile(id, uploadId),
    },
};

/**
 * Admin: add files to a title: episodes to an anime, volumes to a light novel, tracks to an
 * album, or the file of a movie. Each file's details are read from its name and can be
 * corrected first. Files go up one at a time in pieces, so any size gets past the proxy's
 * request limit; if adding one is refused (a number already taken, say) its upload is kept,
 * so fixing the details doesn't send the file again
 * @param {object} dialogRef ref for the <dialog>
 * @param {string} kind episodes, volumes, tracks or movie
 * @param {object} target { id, title } of the anime, light novel, album or movie
 * @param {string[]} have the keys already in the library ("1x5" episodes, "3" volumes,
 *  lower case titles for tracks), so clashes show before uploading
 * @param {string} defaultSeason the season new episode rows start with when the name has none
 * @param {function} onAdded called after uploads finish, to reload the page
 */
export default function UploadFilesModal({ dialogRef, kind, target, have = [], defaultSeason, onAdded }) {
    const config = kinds[kind];
    // { key, file, values, status: ready | uploading | adding | done | error, progress, error, uploadId }
    const [rows, setRows] = useState([]);
    const [running, setRunning] = useState(false);
    const [dragging, setDragging] = useState(false);
    const abortRef = useRef(null);

    const haveSet = new Set(have);

    function addFiles(fileList) {
        const files = [...fileList];
        const accepted = files.filter(f => config.accept.includes(ext(f.name)));
        if (accepted.length < files.length) toast.error(`Only ${config.accept.join(", ")} files can be added here`);
        const picked = config.single ? accepted.slice(0, 1) : accepted;
        setRows(prev => [...(config.single ? [] : prev), ...picked.map(file => ({
            key: nextKey++,
            file,
            values: config.fromName(file.name, { defaultSeason }),
            status: "ready",
            progress: 0,
            error: null,
            uploadId: null,
        }))]);
    }

    function update(key, changes) {
        setRows(prev => prev.map(r => r.key === key ? { ...r, ...changes } : r));
    }

    function setValue(row, field, value) {
        update(row.key, { values: { ...row.values, [field]: value }, status: row.status === "error" ? "ready" : row.status });
    }

    // why a row can't be added yet, or null when it can
    function problem(row) {
        for (const field of config.fields) {
            const v = String(row.values[field.key] ?? "").trim();
            if (field.required && !v) return `Enter the ${field.label.toLowerCase()}`;
            if (v && !field.text && !field.grow && !isWhole(v)) return `The ${field.label.toLowerCase()} must be a whole number`;
        }
        if (config.single) return null;
        const key = config.key(row.values);
        if (haveSet.has(key)) return config.taken;
        if (rows.some(r => r !== row && r.status !== "done" && config.key(r.values) === key)) return "Two files have the same details";
        return null;
    }

    const pending = rows.filter(r => r.status === "ready" || r.status === "error");
    const blocked = pending.some(r => problem(r));
    const done = rows.length > 0 && rows.every(r => r.status === "done");

    async function start() {
        setRunning(true);
        const controller = new AbortController();
        abortRef.current = controller;
        let added = 0;
        for (const row of pending) {
            if (controller.signal.aborted) break;
            let uploadId = row.uploadId;
            try {
                if (!uploadId) {
                    update(row.key, { status: "uploading", progress: 0, error: null });
                    uploadId = await uploadInChunks(row.file, config.uploadKind, p => update(row.key, { progress: p }), controller.signal);
                    update(row.key, { uploadId });
                }
                update(row.key, { status: "adding", progress: 1, error: null });
                await config.add(target.id, uploadId, row.values);
                update(row.key, { status: "done", uploadId: null });
                added++;
            } catch (err) {
                const stopped = controller.signal.aborted;
                update(row.key, { status: "error", error: stopped ? "Stopped" : errorMessage(err, "Couldn't add the file") });
            }
        }
        abortRef.current = null;
        setRunning(false);
        if (added > 0) {
            toast.success(`Added ${added} ${config.noun[added === 1 ? 0 : 1]}`);
            onAdded?.();
        }
    }

    function reset() {
        // uploads kept for a retry that won't happen now
        rows.forEach(r => r.uploadId && cancelUpload(r.uploadId).catch(() => {}));
        setRows([]);
        setDragging(false);
    }

    return createPortal(<dialog className="modal modal-bottom sm:modal-middle" ref={dialogRef} onClose={reset}
        onCancel={e => { if (running) e.preventDefault(); }}>
        <div className="modal-box flex flex-col gap-4 sm:max-w-2xl">
            <div className="flex items-center gap-3">
                <span className="grid size-10 place-items-center rounded-full bg-primary/15 text-primary"><Upload size={18}/></span>
                <div className="min-w-0">
                    <h3 className="text-xl font-semibold">Add {config.single ? "the movie file" : config.noun[1]}</h3>
                    <p className="text-sm text-base-content/50 truncate">{target.title}</p>
                </div>
            </div>

            {!(config.single && rows.length > 0) && <label
                className={`flex flex-col items-center gap-2 rounded-box border-2 border-dashed px-4 py-6 text-center cursor-pointer transition-colors
                    ${dragging ? "border-primary bg-primary/8" : "border-white/15 hover:border-white/30 hover:bg-white/[0.03]"}
                    ${running ? "pointer-events-none opacity-50" : ""}`}
                onDragOver={e => { e.preventDefault(); setDragging(true); }}
                onDragLeave={() => setDragging(false)}
                onDrop={e => { e.preventDefault(); setDragging(false); addFiles(e.dataTransfer.files); }}
            >
                <FileUp size={26} className="text-base-content/40"/>
                <span className="text-sm font-medium">Drop {config.accept.length > 2 ? "video" : config.accept.join(" or ")} {config.single ? "file" : "files"} here, or tap to choose</span>
                <span className="text-xs text-base-content/45">{config.hint}</span>
                <input type="file" multiple={!config.single} accept={config.accept.join(",")} className="hidden"
                    onChange={e => { addFiles(e.target.files); e.target.value = ""; }}/>
            </label>}

            {rows.length > 0 && <ul className="flex flex-col gap-1.5 max-h-[45dvh] overflow-y-auto -mx-1 px-1">
                {rows.map(row => {
                    const editable = row.status === "ready" || row.status === "error";
                    const issue = editable ? problem(row) : null;
                    return <li key={row.key} className="rounded-field bg-white/[0.03] ring-1 ring-white/5 px-3 py-2.5 flex flex-col gap-2">
                        {config.fields.length > 0 && <div className="flex flex-wrap items-center gap-2">
                            {config.fields.map(field => <label key={field.key} className={`flex items-center gap-1.5 ${field.grow ? "flex-1 min-w-40" : "shrink-0"}`}>
                                {field.short && <span className="text-xs text-base-content/45">{field.short}</span>}
                                <input
                                    className={`input input-sm bg-base-100/60 text-base sm:text-sm ${field.grow ? "w-full" : `${field.width} tabular-nums`}
                                        ${issue && field.required && !String(row.values[field.key] ?? "").trim() ? "border-error/60" : "border-white/10"}`}
                                    value={row.values[field.key] ?? ""}
                                    inputMode={field.grow || field.text ? "text" : "numeric"}
                                    placeholder={field.grow ? field.label : ""}
                                    disabled={!editable || running}
                                    onChange={e => setValue(row, field.key, e.target.value)}
                                    aria-label={`${field.label} for ${row.file.name}`}
                                />
                            </label>)}
                        </div>}
                        <div className="flex items-center gap-3">
                            <div className="flex-1 min-w-0">
                                <p className="text-sm truncate" title={row.file.name}>{row.file.name}</p>
                                <p className={`text-xs truncate ${issue || row.error ? "text-error" : "text-base-content/45"}`}>
                                    {row.error ?? issue ?? (row.status === "adding" ? "Adding to the library…" : readableSize(row.file.size))}
                                </p>
                            </div>
                            {row.status === "done" ? <span className="grid size-7 place-items-center rounded-full bg-success/15 text-success"><Check size={15}/></span>
                                : row.status === "uploading" ? <span className="text-xs tabular-nums text-base-content/60 w-10 text-right">{Math.floor(row.progress * 100)}%</span>
                                : row.status === "adding" ? <span className="loading loading-spinner loading-xs"/>
                                : row.status === "error" ? <AlertCircle size={18} className="text-error shrink-0"/>
                                : <button className="btn btn-ghost btn-xs btn-square" disabled={running}
                                    onClick={() => setRows(prev => prev.filter(r => r.key !== row.key))} aria-label={`Remove ${row.file.name}`}>
                                    <X size={14}/>
                                </button>}
                        </div>
                        {row.status === "uploading" && <div className="h-1 rounded-full bg-white/10 overflow-hidden">
                            <div className="h-full bg-primary transition-[width] duration-200" style={{ width: `${Math.round(row.progress * 100)}%` }}></div>
                        </div>}
                    </li>
                })}
            </ul>}

            <div className="flex flex-wrap items-center justify-end gap-2 pt-1">
                {pending.length > 0 && !running && !config.single && <span className="mr-auto text-sm text-base-content/50">
                    {pending.length} {config.noun[pending.length === 1 ? 0 : 1]} to add
                </span>}
                {running && <span className="mr-auto text-xs text-base-content/50">Keep this open until the uploads finish</span>}
                {running
                    ? <button className="btn btn-ghost" onClick={() => abortRef.current?.abort()}>Stop</button>
                    : <button className="btn btn-ghost" onClick={() => dialogRef.current.close()}>{done ? "Close" : "Cancel"}</button>}
                {!done && <button className="btn btn-primary" onClick={start} disabled={running || pending.length === 0 || blocked}>
                    {running ? <span className="loading loading-spinner loading-xs"/> : <Upload size={16}/>}
                    {running ? "Uploading…" : "Upload"}
                </button>}
            </div>
        </div>
    </dialog>, document.body)
}

function readableSize(bytes) {
    if (bytes >= 1024 ** 3) return `${(bytes / 1024 ** 3).toFixed(2)} GB`;
    return `${(bytes / 1024 ** 2).toFixed(1)} MB`;
}
