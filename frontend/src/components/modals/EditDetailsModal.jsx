import { useState } from "react";
import { createPortal } from "react-dom";
import { ImagePlus, PencilLine, RotateCcw, Undo2 } from "lucide-react";
import Cover from "../Cover";
import TextField from "../TextField";
import { toast } from "../../stores/useToastStore";
import { editAlbum, editMedia, uploadAlbumCover, uploadMediaCover } from "../../services/curationService";
import {
    changedFields, dateInputValue, descriptionToText, genresFromText, textToDescription,
} from "../../utils/curation";

// the fields each type has beyond title, description, release date and genres
const typeFields = {
    anime: [
        { key: "studio", label: "Studio" },
        { key: "status", label: "Status", options: ["airing", "finished", "upcoming"] },
    ],
    movie: [
        { key: "director", label: "Director" },
        { key: "runtime_mins", label: "Runtime (minutes)", number: true },
    ],
    manga: [
        { key: "status", label: "Status", options: ["ongoing", "completed", "hiatus"] },
        { key: "total_chapters", label: "Total chapters", number: true },
    ],
    light_novel: [
        { key: "author", label: "Author" },
        { key: "total_volumes", label: "Total volumes", number: true },
    ],
    music_track: [
        { key: "artist", label: "Artist" },
        { key: "track_number", label: "Track number", number: true },
    ],
    music_album: [
        { key: "artist", label: "Artist" },
    ],
};

// albums keep no genres
const typeHasGenres = type => type !== "music_album";

// a replaced cover keeps its URL, so a version makes the browser load the new one
const freshURL = url => `${url}&v=${Date.now()}`;

/**
 * Admin: edit a title's details and cover. Fields the admin changed are kept through library
 * syncs; each one shows "Edited" with a way to hand it back to the syncs
 * @param {object} dialogRef ref for the <dialog>
 * @param {object} item the media item as GET /media/:id returns it, or an album with
 *  type "music_album" (GET /albums/:id)
 * @param {boolean} html the description is shown as HTML (AniList's descriptions are)
 * @param {function} onSaved called after saving, with { cover } set to a fresh URL when the
 *  cover changed
 */
export default function EditDetailsModal({ dialogRef, item, html = false, onSaved }) {
    // the form is rebuilt from the item each time the dialog opens
    const [session, setSession] = useState(0);

    return createPortal(
        <dialog className="modal modal-bottom sm:modal-middle" ref={dialogRef} onClose={() => setSession(s => s + 1)}>
            {item && <EditDetailsForm key={session} dialogRef={dialogRef} item={item} html={html} onSaved={onSaved}/>}
            <form method="dialog" className="modal-backdrop"><button>close</button></form>
        </dialog>,
        document.body
    );
}

function initialValues(item, html) {
    const meta = item.metadata ?? {};
    const values = {
        title: item.title ?? "",
        release_date: dateInputValue(item.release_date),
        description: descriptionToText(item.description, html),
    };
    if (typeHasGenres(item.type)) values.genres = (meta.genres ?? []).join(", ");
    for (const field of typeFields[item.type] ?? []) {
        const value = item.type === "music_album" ? item[field.key] : meta[field.key];
        values[field.key] = value === null || value === undefined ? "" : String(value);
    }
    return values;
}

function EditDetailsForm({ dialogRef, item, html, onSaved }) {
    const isAlbum = item.type === "music_album";
    const [initial] = useState(() => initialValues(item, html));
    const [values, setValues] = useState(initial);
    const [unlock, setUnlock] = useState([]);
    const [coverFile, setCoverFile] = useState(null);
    const [coverPreview, setCoverPreview] = useState(null);
    const [saving, setSaving] = useState(false);
    const locked = new Set(item.locked_fields ?? []);
    const fields = typeFields[item.type] ?? [];

    const set = key => e => setValues(v => ({ ...v, [key]: e.target.value }));
    const toggleUnlock = key => setUnlock(u => u.includes(key) ? u.filter(k => k !== key) : [...u, key]);

    function pickCover(e) {
        const file = e.target.files?.[0];
        e.target.value = "";
        if (!file) return;
        if (coverPreview) URL.revokeObjectURL(coverPreview);
        setCoverFile(file);
        setCoverPreview(URL.createObjectURL(file));
        setUnlock(u => u.filter(k => k !== "cover"));
    }

    async function save(e) {
        e.preventDefault();
        if (!values.title.trim()) return toast.error("The title can't be empty");

        const numberFields = fields.filter(f => f.number).map(f => f.key);
        const changes = changedFields(initial, values, numberFields);
        if ("genres" in changes) changes.genres = genresFromText(values.genres);
        if ("description" in changes) changes.description = textToDescription(values.description, html);
        // a field handed back keeps the value the sync gives it, not one typed here
        const handBack = unlock.filter(k => !(k in changes));

        if (Object.keys(changes).length === 0 && handBack.length === 0 && !coverFile) {
            dialogRef.current.close();
            return;
        }

        setSaving(true);
        try {
            if (isAlbum) {
                if (Object.keys(changes).length > 0) await editAlbum(item.id, changes);
            } else if (Object.keys(changes).length > 0 || handBack.length > 0) {
                await editMedia(item.id, changes, handBack);
            }
            let cover;
            if (coverFile) {
                const res = await (isAlbum ? uploadAlbumCover : uploadMediaCover)(item.id, coverFile);
                cover = freshURL(res.data.cover_image_url);
            }
            toast.success("Saved");
            dialogRef.current.close();
            onSaved?.({ cover, coverCleared: handBack.includes("cover") });
        } catch (err) {
            toast.error(err, "Could not save the changes");
        } finally {
            setSaving(false);
        }
    }

    const lockProps = key => isAlbum ? {} : { locked: locked.has(key), handedBack: unlock.includes(key), onToggle: () => toggleUnlock(key) };
    const coverShown = coverPreview ?? (unlock.includes("cover") ? null : item.cover_image_url);

    return <form className="modal-box flex flex-col gap-5 sm:max-w-3xl" onSubmit={save}>
        <div className="flex items-center gap-3">
            <span className="grid size-10 place-items-center rounded-full bg-primary/15 text-primary"><PencilLine size={18}/></span>
            <div className="min-w-0">
                <h3 className="text-xl font-semibold">{isAlbum ? "Edit album" : "Edit details"}</h3>
                <p className="text-sm text-base-content/50 truncate">{item.title}</p>
            </div>
        </div>

        <div className="flex flex-col sm:flex-row gap-5">
            <div className="flex sm:flex-col items-center sm:items-stretch gap-4 sm:gap-3 sm:w-40 shrink-0">
                <div className={`overflow-hidden rounded-box bg-base-300 ring-1 ring-white/10 shrink-0
                    ${isAlbum ? "w-24 sm:w-40 aspect-square" : "w-24 sm:w-40 aspect-[2/3]"}`}>
                    <Cover src={coverShown} source={coverPreview ? undefined : item.external_source} alt="Cover" className="size-full object-cover"/>
                </div>
                <div className="flex flex-col gap-2 min-w-0">
                    <label className="btn btn-sm border-white/10 bg-white/10 hover:bg-white/15">
                        <ImagePlus size={15}/> {coverFile ? "Pick another" : "Change cover"}
                        <input type="file" accept="image/jpeg,image/png,image/webp,image/gif" className="hidden" onChange={pickCover}/>
                    </label>
                    {coverFile && <span className="text-xs text-base-content/50 truncate" title={coverFile.name}>{coverFile.name}</span>}
                    {!isAlbum && !coverFile && locked.has("cover") && <HandBack {...lockProps("cover")} label="cover"/>}
                </div>
            </div>

            <div className="flex-1 min-w-0 flex flex-col gap-4">
                <Field label="Title" {...lockProps("title")}>
                    <TextField value={values.title} onChange={set("title")} required maxLength={300} aria-label="Title"/>
                </Field>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    {fields.map(field => <Field key={field.key} label={field.label} {...lockProps(field.key)}>
                        {field.options
                            ? <select className="select w-full h-11 bg-base-100/60 border-white/10 text-base sm:text-sm"
                                value={values[field.key]} onChange={set(field.key)} aria-label={field.label}>
                                <option value="">Unknown</option>
                                {field.options.map(o => <option key={o} value={o}>{o[0].toUpperCase() + o.slice(1)}</option>)}
                            </select>
                            : <TextField value={values[field.key]} onChange={set(field.key)} aria-label={field.label}
                                {...(field.number ? { type: "number", min: 0, inputMode: "numeric" } : { maxLength: 300 })}/>}
                    </Field>)}
                    <Field label="Release date" {...lockProps("release_date")}>
                        <TextField type="date" value={values.release_date} onChange={set("release_date")} aria-label="Release date"/>
                    </Field>
                </div>

                {typeHasGenres(item.type) && <Field label="Genres" hint="Separate with commas" {...lockProps("genres")}>
                    <TextField value={values.genres} onChange={set("genres")} placeholder="Action, Fantasy" aria-label="Genres"/>
                </Field>}

                <Field label="Description" {...lockProps("description")}>
                    <textarea
                        className="textarea w-full min-h-36 bg-base-100/60 border-white/10 focus:border-primary/60 focus:outline-none text-base sm:text-sm leading-relaxed"
                        value={values.description}
                        onChange={set("description")}
                        maxLength={20000}
                        aria-label="Description"
                    />
                </Field>
            </div>
        </div>

        <div className="flex flex-wrap items-center justify-end gap-2 pt-1">
            {!isAlbum && <p className="mr-auto text-xs text-base-content/45 max-w-sm">
                What you change here stays when the library syncs.
            </p>}
            <button type="button" className="btn btn-ghost" onClick={() => dialogRef.current.close()} disabled={saving}>Cancel</button>
            <button type="submit" className="btn btn-primary" disabled={saving}>
                {saving && <span className="loading loading-spinner loading-xs"/>} Save
            </button>
        </div>
    </form>
}

/**
 * A form field with its label, and when an admin set it before, an Edited marker that can
 * hand the field back to the library syncs
 */
function Field({ label, hint, locked, handedBack, onToggle, children }) {
    return <div className="flex flex-col gap-1.5">
        <div className="flex items-center justify-between gap-2 min-h-5">
            <span className="text-xs font-medium uppercase tracking-wider text-base-content/55">{label}</span>
            {locked && <HandBack locked handedBack={handedBack} onToggle={onToggle} label={label.toLowerCase()} compact/>}
        </div>
        {children}
        {handedBack
            ? <span className="text-xs text-warning/90">Goes back to the synced value on the next library sync</span>
            : hint && <span className="text-xs text-base-content/45">{hint}</span>}
    </div>
}

function HandBack({ locked, handedBack, onToggle, label, compact = false }) {
    if (!locked) return null;
    if (handedBack) return <button type="button" className="inline-flex items-center gap-1 text-xs text-base-content/60 hover:text-base-content" onClick={onToggle}>
        <Undo2 size={12}/> Keep my {label}
    </button>;
    return <span className="inline-flex items-center gap-1.5 text-xs">
        {compact && <span className="rounded-full bg-primary/15 text-primary px-2 py-px font-medium">Edited</span>}
        <button type="button" className="inline-flex items-center gap-1 text-base-content/50 hover:text-base-content" onClick={onToggle}
            title={`Let the library sync set the ${label} again`}>
            <RotateCcw size={12}/> {compact ? "Use synced" : `Use the synced ${label}`}
        </button>
    </span>
}
