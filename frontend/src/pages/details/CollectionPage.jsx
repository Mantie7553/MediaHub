import { useRef, useState } from "react"
import { Link, useNavigate, useParams } from "react-router-dom"
import { ArrowDown, ArrowUp, ArrowUpToLine, Check, Clock, Copy, GripVertical, ListPlus, PencilLine, Send, SlidersHorizontal, Sparkles, Trash2, X } from "lucide-react"
import useApi from "../../hooks/useApi"
import useCollectionPicker from "../../hooks/useCollectionPicker"
import { useUserContent } from "../../hooks"
import useAudioStore from "../../stores/useAudioStore"
import { toast } from "../../stores/useToastStore"
import DetailsHero from "../../components/details/DetailsHero"
import TrackList from "../../components/details/TrackList"
import { ActionMenu, ActionMenuItem } from "../../components/details/Menus"
import { AlbumCard, Card, CollectionCard, CollectionCover } from "../../components/cards"
import { cardGrid } from "../../components/layout/cardGrid"
import Cover from "../../components/Cover"
import ConfirmModal from "../../components/modals/ConfirmModal"
import CollectionNameModal from "../../components/modals/CollectionNameModal"
import NoteModal from "../../components/modals/NoteModal"
import EmptyState from "../../components/states/EmptyState"
import ErrorState from "../../components/states/ErrorState"
import NotFound from "../../components/states/NotFound"
import { DetailsSkeleton } from "../../components/states/Skeletons"
import { deleteCollection, featureNow, moveInCollection, removeFromCollection, saveCopy, submitForFeaturing, updateCollection, withdrawSubmission } from "../../services/collectionsService"
import { getRole } from "../../utils/auth"
import { SMART_PLAYLISTS, applyMove, dropTarget, moveTarget } from "../../utils/collections"
import { PlayButton, QueueModeButtons } from "../../components/details/PlaybackControls"

// what each kind of entry is called on its own line
const typeLabels = {
    anime: "Anime", movie: "Movie", manga: "Manga", light_novel: "Light novel",
    music_track: "Track", album: "Album", playlist: "Playlist",
}

/**
 * A playlist (/playlists/:id) or collection (/collections/:id): play it, change what's in it and
 * its order, rename or delete it. The two playlists that fill themselves (listened, unlistened)
 * can be played but not changed. Owners can submit theirs to be featured; a featured one can't be
 * changed, but anyone can save a copy of it, and admins can take it down
 */
export default function CollectionPage() {
    const { id } = useParams()
    const navigate = useNavigate()
    const { data, loading, error, notFound, reload } = useApi(`/collections/${id}`)
    const { userContentMap, refresh } = useUserContent()
    const playAlbum = useAudioStore(state => state.playAlbum)
    const picker = useCollectionPicker()
    const renameRef = useRef(null)
    const deleteRef = useRef(null)
    const takeDownRef = useRef(null)
    const [arranging, setArranging] = useState(false)
    const [copying, setCopying] = useState(false)

    // the order shown, changed straight away when something moves and put back from the server
    // whenever a fresh copy loads
    const [entries, setEntries] = useState(data?.items ?? [])
    const [shownData, setShownData] = useState(data)
    if (data !== shownData) {
        setShownData(data)
        setEntries(data?.items ?? [])
    }

    if (loading) return <DetailsSkeleton square/>
    if (notFound) return <NotFound title="Not found" hint="This playlist or collection doesn't exist, isn't yours, or was taken down."/>
    if (error) return <ErrorState error={error} title="Couldn't load this" onRetry={reload} showBack/>
    if (!data) return null

    const isPlaylist = data.kind === "playlist"
    const noun = isPlaylist ? "playlist" : "collection"
    const canEdit = data.can_edit
    const featured = data.featured

    function move(index, after) {
        if (after === undefined) return
        const entry = entries[index]
        setEntries(applyMove(entries, index, after))
        moveInCollection(id, entry.entry_id, after)
            .catch(err => { toast.error(err, "Could not move it"); reload(true) })
    }

    function remove(entry) {
        setEntries(entries.filter(e => e.entry_id !== entry.entry_id))
        removeFromCollection(id, entry.entry_id)
            .then(() => { toast.success(`Removed ${entry.title}`); reload(true) })
            .catch(err => { toast.error(err, "Could not remove it"); reload(true) })
    }

    function rename(changes) {
        return updateCollection(id, changes).then(() => reload(true))
    }

    function destroy() {
        return deleteCollection(id)
            .then(() => { toast.success(featured ? `Took down ${data.name}` : `Deleted ${data.name}`); navigate("/", { replace: true }) })
            .catch(err => { toast.error(err, `Could not delete the ${noun}`); throw err })
    }

    function copyToMine() {
        setCopying(true)
        saveCopy(id)
            .then(res => {
                toast.success(`Saved a copy of ${data.name} to your ${noun}s`)
                navigate(`/${res.data.kind === "collection" ? "collections" : "playlists"}/${res.data.id}`)
            })
            .catch(err => toast.error(err, "Could not save a copy"))
            .finally(() => setCopying(false))
    }

    // the menu items that move or remove an entry, for the track rows
    const editItems = (entry, i) => canEdit && <>
        <ActionMenuItem icon={ArrowUpToLine} label="Move to top" onClick={() => move(i, moveTarget(entries, i, "top"))}/>
        <ActionMenuItem icon={ArrowUp} label="Move up" onClick={() => move(i, moveTarget(entries, i, "up"))}/>
        <ActionMenuItem icon={ArrowDown} label="Move down" onClick={() => move(i, moveTarget(entries, i, "down"))}/>
        <ActionMenuItem icon={Trash2} label={`Remove from ${noun}`} danger onClick={() => remove(entry)}/>
    </>

    const playable = entries.filter(e => e.type === "music_track" && e.available)
    const queue = playable.map(t => ({ id: t.id, title: t.title, artist: t.artist, thumbnail: t.cover_image_url }))
    const totalSecs = entries.reduce((sum, e) => sum + (e.type === "music_track" ? e.duration_secs ?? 0 : 0), 0)
    const count = entries.length
    const description = data.smart ? SMART_PLAYLISTS[data.id] : data.description

    return <div className="flex flex-col gap-8">
        <DetailsHero
            cover={{ src: data.covers?.[0] }}
            square
            poster={<CollectionCover covers={data.covers} kind={data.kind} smart={data.smart} className="size-full"/>}
            eyebrow={data.smart ? "Playlist · fills itself" : featured ? `Featured ${noun}` : isPlaylist ? "Playlist" : "Collection"}
            title={data.name}
            meta={[
                `${count} ${isPlaylist ? (count === 1 ? "track" : "tracks") : (count === 1 ? "item" : "items")}`,
                totalSecs > 0 && `${Math.round(totalSecs / 60)} min`,
            ]}
            description={description}
            primary={isPlaylist
                ? <PlayButton tracks={queue} source={`/playlists/${id}`}/>
                : featured ? <button className="btn btn-lg px-6 border-white/10 bg-white/10 hover:bg-white/15" onClick={copyToMine} disabled={copying}>
                    {copying ? <span className="loading loading-spinner loading-sm"/> : <Copy size={18}/>} Save a copy
                </button>
                : canEdit && count > 0 && <button className={`btn btn-lg px-6 ${arranging ? "btn-primary" : "border-white/10 bg-white/10 hover:bg-white/15"}`}
                    onClick={() => setArranging(!arranging)}>
                    {arranging ? <><Check size={18}/> Done</> : <><SlidersHorizontal size={18}/> Arrange</>}
                </button>}
            actions={<>
                {isPlaylist && <QueueModeButtons/>}
                {(canEdit || featured || (isPlaylist && !data.smart)) && <ActionMenu label={`${data.name} actions`}>
                    {isPlaylist && featured && <ActionMenuItem icon={Copy} label="Save a copy to my playlists" onClick={copyToMine}/>}
                    {isPlaylist && !data.smart && <ActionMenuItem icon={ListPlus} label="Add to a collection…"
                        onClick={() => picker.open({ id, kind: "playlist", title: data.name })}/>}
                    {data.can_delete && featured && <ActionMenuItem icon={Trash2} label="Take down…" danger onClick={() => takeDownRef.current.showModal()}/>}
                    {canEdit && <>
                        <ActionMenuItem icon={PencilLine} label={`Rename ${noun}…`} onClick={() => renameRef.current.showModal()}/>
                        <ActionMenuItem icon={Trash2} label={`Delete ${noun}…`} danger onClick={() => deleteRef.current.showModal()}/>
                    </>}
                </ActionMenu>}
            </>}
        />

        {canEdit && !data.smart && count > 0 && <FeatureStatus collection={data} noun={noun} onChange={() => reload(true)}/>}
        {!canEdit && !featured && data.in_review && <ReviewNotice/>}

        {count === 0 ? (
            <EmptyState compact
                title={data.smart ? "Nothing here yet" : `This ${noun} is empty`}
                hint={data.smart
                    ? "Add tracks or albums to your list as listened, or to listen to, and they show up here."
                    : isPlaylist
                        ? "Add tracks or whole albums with “Add to playlist” on an album page."
                        : "Add titles, albums, tracks and playlists with “Add to a collection”."}
                action={{ label: "Go to the dashboard", to: "/" }}
            />
        ) : isPlaylist ? (
            <TrackList
                tracks={entries.map(e => ({
                    key: e.entry_id || e.id, id: e.id, title: e.title, available: e.available,
                    subtitle: [e.artist, e.album_title].filter(Boolean).join(" · "), duration_secs: e.duration_secs,
                }))}
                onPlay={i => playAlbum(queue, playable.indexOf(entries[i]), `/playlists/${id}`)}
                menu={(track, i) => <ActionMenu label={`${track.title} actions`} size="sm">
                    <ActionMenuItem icon={ListPlus} label="Add to playlist…" onClick={() => picker.open({ id: track.id, type: "music_track", title: track.title })}/>
                    {editItems(entries[i], i)}
                </ActionMenu>}
                onReorder={canEdit ? (from, to) => move(from, dropTarget(entries, from, to)) : undefined}
            />
        ) : arranging ? (
            <ArrangeList entries={entries} onMove={move} onRemove={remove}/>
        ) : (
            <ul className={cardGrid}>
                {entries.map(e => <EntryCard key={e.entry_id} entry={e} userContentMap={userContentMap} onListChange={refresh}/>)}
            </ul>
        )}

        {canEdit && <>
            <CollectionNameModal dialogRef={renameRef} kind={data.kind} initial={{ name: data.name, description: data.description ?? "" }} onSave={rename}/>
            <ConfirmModal dialogRef={deleteRef} title={`Delete “${data.name}”?`} confirmLabel={`Delete ${noun}`}
                lines={[
                    `The ${noun} goes away for good. What's in it stays in the library.`,
                    ...(isPlaylist ? ["Collections it's in lose it too."] : []),
                ]}
                onConfirm={destroy}/>
        </>}
        {data.can_delete && featured && <ConfirmModal dialogRef={takeDownRef} title={`Take down “${data.name}”?`} confirmLabel="Take down"
            lines={[
                "It leaves everyone's Featured row for good. Copies people saved are theirs and stay.",
                "The original it was made from isn't touched, so it can be submitted again.",
            ]}
            onConfirm={destroy}/>}
        {picker.modal}
    </div>
}

/**
 * For the owner: where featuring this has got to, and the button that asks for it (or, for
 * admins, does it). One that is already featured sends its changes as an update to the featured
 * copy
 */
function FeatureStatus({ collection, noun, onChange }) {
    const submitRef = useRef(null)
    const [busy, setBusy] = useState(false)
    const isAdmin = getRole() === "admin"
    const submission = collection.submission
    const copyId = collection.featured_copy_id
    const copyPath = copyId && `/${collection.kind === "collection" ? "collections" : "playlists"}/${copyId}`
    const waiting = submission?.status === "pending"
    const rejected = submission?.status === "rejected"

    function submit(note) {
        return submitForFeaturing(collection.id, note).then(() => {
            toast.success(copyId ? "Sent your changes for review" : `Sent your ${noun} for review`)
            onChange()
        })
    }

    function withdraw() {
        setBusy(true)
        withdrawSubmission(collection.id)
            .then(() => { toast.success("Took back your request"); onChange() })
            .catch(err => toast.error(err, "Could not take it back"))
            .finally(() => setBusy(false))
    }

    function featureItNow() {
        setBusy(true)
        featureNow(collection.id)
            .then(() => { toast.success(copyId ? "Updated the featured copy" : `Featured your ${noun}`); onChange() })
            .catch(err => toast.error(err, `Could not feature the ${noun}`))
            .finally(() => setBusy(false))
    }

    let Icon = Sparkles
    let text = copyId
        ? "A copy of this is featured. Changes you make here stay yours until you send them."
        : "Share it with everyone: once an admin approves it, a copy shows in everyone's Featured row."
    if (waiting) {
        Icon = Clock
        text = copyId ? "Your changes are waiting for an admin's review." : "Waiting for an admin's review."
    } else if (rejected) {
        Icon = X
        text = <>An admin didn't feature this{submission.admin_note
            ? <>: <span className="text-base-content/90">“{submission.admin_note}”</span></>
            : "."} You can change it and send it again.</>
    }

    return <section className="surface flex flex-wrap items-center gap-x-4 gap-y-3 px-4 py-3 sm:px-5 animate-fade-up">
        <Icon size={18} className={`shrink-0 ${rejected ? "text-error" : "text-primary"}`}/>
        <p className="flex-1 min-w-48 text-sm text-base-content/70">{text}</p>
        <div className="flex flex-wrap gap-2">
            {copyPath && <Link to={copyPath} className="btn btn-sm btn-ghost">View featured</Link>}
            {waiting && <button className="btn btn-sm btn-ghost" onClick={withdraw} disabled={busy}>Take back</button>}
            {isAdmin
                ? <button className="btn btn-sm btn-primary" onClick={featureItNow} disabled={busy}>
                    {busy ? <span className="loading loading-spinner loading-xs"/> : <Sparkles size={15}/>}
                    {copyId ? "Update featured copy" : "Feature now"}
                </button>
                : !waiting && <button className="btn btn-sm btn-primary" onClick={() => submitRef.current.showModal()}>
                    <Send size={15}/> {copyId ? "Send changes" : "Submit for featuring"}
                </button>}
        </div>
        <NoteModal dialogRef={submitRef}
            title={copyId ? "Send your changes for review" : `Submit “${collection.name}” for featuring`}
            intro={copyId
                ? `If an admin approves, the featured copy is replaced with your ${noun} as it is then.`
                : `An admin looks it over. If they approve, a copy of your ${noun} as it is then shows for everyone; later changes stay yours until you send them.`}
            label="Message for the admins (optional)"
            confirmLabel="Send for review"
            onSave={submit}/>
    </section>
}

/**
 * For an admin opening someone's playlist or collection from the review queue
 */
function ReviewNotice() {
    return <section className="surface flex flex-wrap items-center gap-x-4 gap-y-3 px-4 py-3 sm:px-5 animate-fade-up">
        <Clock size={18} className="shrink-0 text-primary"/>
        <p className="flex-1 min-w-48 text-sm text-base-content/70">Someone asked for this to be featured. Approve or turn it down from Downloads.</p>
        <Link to="/downloads" className="btn btn-sm btn-ghost">Go to review</Link>
    </section>
}

/**
 * One entry of a collection as a card: titles and tracks as posters, albums and playlists square
 */
function EntryCard({ entry, userContentMap, onListChange }) {
    if (entry.type === "album") {
        return <AlbumCard fluid album={{ id: entry.id, title: entry.title, artist: entry.artist, cover_image_url: entry.cover_image_url, track_count: entry.item_count }}/>
    }
    if (entry.type === "playlist") {
        return <CollectionCard fluid collection={{ id: entry.id, kind: "playlist", name: entry.title, item_count: entry.item_count ?? 0, covers: entry.covers ?? [] }}/>
    }
    return <Card fluid userContentMap={userContentMap} onListChange={onListChange}
        item={{
            id: entry.id, type: entry.type, title: entry.title, artist: entry.artist,
            cover_image_url: entry.cover_image_url, release_date: entry.release_date, external_source: entry.external_source,
        }}/>
}

/**
 * A collection's entries as rows to put in order or take out: buttons on every row, and
 * dragging with a mouse
 */
function ArrangeList({ entries, onMove, onRemove }) {
    const [dragging, setDragging] = useState(null)
    const [over, setOver] = useState(null)

    return <section className="surface p-1.5 sm:p-2 animate-fade-up">
        <ol className="flex flex-col gap-0.5">
            {entries.map((e, i) => {
                const marker = over === i && dragging !== null && dragging !== i
                    ? (dragging > i ? "shadow-[inset_0_2px_0_0_var(--color-primary)]" : "shadow-[inset_0_-2px_0_0_var(--color-primary)]")
                    : ""
                return <li key={e.entry_id}
                    className={`flex items-center gap-3 rounded-field px-2 py-1.5 ${marker} ${dragging === i ? "opacity-40" : ""}`}
                    draggable
                    onDragStart={ev => { setDragging(i); ev.dataTransfer.effectAllowed = "move"; ev.dataTransfer.setData("text/plain", String(i)) }}
                    onDragOver={ev => { ev.preventDefault(); if (over !== i) setOver(i) }}
                    onDragEnd={() => { setDragging(null); setOver(null) }}
                    onDrop={ev => { ev.preventDefault(); if (dragging !== null) onMove(dragging, dropTarget(entries, dragging, i)); setDragging(null); setOver(null) }}
                >
                    <span className="hidden pointer-fine:flex text-base-content/25 cursor-grab" aria-hidden="true"><GripVertical size={16}/></span>
                    {e.type === "playlist"
                        ? <CollectionCover covers={e.covers ?? []} kind="playlist" className="size-11 shrink-0 rounded-md overflow-hidden"/>
                        : <Cover src={e.cover_image_url} source={e.external_source} alt="" className="size-11 shrink-0 rounded-md object-cover"/>}
                    <span className="flex-1 min-w-0">
                        <span className="block truncate text-sm font-medium">{e.title}</span>
                        <span className="block truncate text-xs text-base-content/50">{[typeLabels[e.type] ?? e.type, e.artist].filter(Boolean).join(" · ")}</span>
                    </span>
                    <span className="flex items-center gap-0.5 shrink-0">
                        <button className="btn btn-ghost btn-sm btn-square" disabled={i === 0} onClick={() => onMove(i, moveTarget(entries, i, "up"))} aria-label={`Move ${e.title} up`}>
                            <ArrowUp size={16}/>
                        </button>
                        <button className="btn btn-ghost btn-sm btn-square" disabled={i === entries.length - 1} onClick={() => onMove(i, moveTarget(entries, i, "down"))} aria-label={`Move ${e.title} down`}>
                            <ArrowDown size={16}/>
                        </button>
                        <button className="btn btn-ghost btn-sm btn-square text-error" onClick={() => onRemove(e)} aria-label={`Remove ${e.title}`}>
                            <X size={16}/>
                        </button>
                    </span>
                </li>
            })}
        </ol>
    </section>
}
