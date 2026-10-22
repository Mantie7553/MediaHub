import { useState, useRef } from "react";
import { useNavigate, useParams } from "react-router-dom";
import ErrorState from "../../components/states/ErrorState";
import NotFound from "../../components/states/NotFound";
import EmptyState from "../../components/states/EmptyState";
import { DetailsSkeleton } from "../../components/states/Skeletons";
import { toast } from "../../stores/useToastStore";
import { openViewer } from "../../utils/navigation";
import { useMediaItem, useUserContent, useRemoveFromList } from "../../hooks";
import DeleteFromServerButton from "../../components/DeleteFromServerButton";
import api from "../../services/api";
import { BookOpen, CheckCheck, PencilLine, Undo2, Upload, ListPlus } from "lucide-react";
import Format from "../../utils/format";
import DetailsHero from "../../components/details/DetailsHero";
import { ActionMenu, ActionMenuItem, ListStatusMenu } from "../../components/details/Menus";
import ItemRow, { ItemList } from "../../components/details/ItemRow";
import EditDetailsModal from "../../components/modals/EditDetailsModal";
import UploadFilesModal from "../../components/modals/UploadFilesModal";
import useAdminPieces from "../../hooks/useAdminPieces";
import useCollectionPicker from "../../hooks/useCollectionPicker";
import { getRole } from "../../utils/auth";

export default function LightNovelDetailsPage() {
    const navigate = useNavigate();
    const { userContentMap, refresh } = useUserContent();
    const { id } = useParams();
    const { item: ln, loading, error, notFound, refetch } = useMediaItem(id);
    const userEntry = userContentMap[id];
    const removeEntry = useRemoveFromList(userEntry, refresh);
    const [readVolumes, setReadVolumes] = useState(new Set());
    const [scrollPosition, setScrollPosition] = useState({});
    const editRef = useRef(null);
    const uploadRef = useRef(null);
    // a cover just uploaded, with a version so the browser doesn't show the one it cached
    const [newCover, setNewCover] = useState(null);
    const isAdmin = getRole() === "admin";
    const pieces = useAdminPieces(() => refetch(false));
    const picker = useCollectionPicker();

    // read state and positions start from each copy of the title that loads
    const [shownLn, setShownLn] = useState(null);
    if (ln !== shownLn) {
        setShownLn(ln);
        if (ln?.metadata?.volumes) {
            setReadVolumes(new Set(ln.metadata.volumes.filter(v => v.completed).map(v => v.id)));
            const initial = {};
            ln.metadata.volumes.forEach(v => { initial[v.id] = v.scroll_position ?? 0; });
            setScrollPosition(initial);
        }
    }

    function updateStatus(status) {
        const request = userEntry
            ? api.put(`/me/media/${userEntry.id}`, { status })
            : api.post(`/me/media`, { media_item_id: id, status });
        request
            .then(() => refresh())
            .catch(err => toast.error(err, "Could not update your list"));
    }

    function toggleVolume(volume) {
        const read = !readVolumes.has(volume.id);
        api.put(`/light-novels/volumes/${volume.id}/read`, { read })
            .then(() => {
                const next = new Set(readVolumes);
                read ? next.add(volume.id) : next.delete(volume.id);
                const status = next.size === ln.metadata.volumes.length ? "completed" : "current";
                setReadVolumes(next);
                updateStatus(status);
            })
            .catch(err => toast.error(err, "Could not update the volume"));
        api.put(`/light-novels/volumes/${volume.id}/progress`, { scroll_position: read ? 1 : 0 })
            .then(() => setScrollPosition(prev => ({ ...prev, [volume.id]: read ? 1 : 0 })))
            // the read mark above already reports a failure
            .catch(() => {});
    }

    function markAll(read) {
        api.put(`/light-novels/${id}/read`, { read })
            .then(() => {
                setReadVolumes(read ? new Set(ln.metadata.volumes.map(v => v.id)) : new Set());
                updateStatus(read ? "completed" : "planned");
                const newScrollPositions = {};
                ln.metadata.volumes.forEach(v => { newScrollPositions[v.id] = read ? 1 : 0; });
                setScrollPosition(newScrollPositions);
                Promise.all(ln.metadata.volumes.map(v =>
                    api.put(`/light-novels/volumes/${v.id}/progress`, { scroll_position: read ? 1 : 0 })
                )).catch(err => toast.error(err, "Some volumes' reading progress didn't save"));
            })
            .catch(err => toast.error(err, "Could not update the volumes"));
    }

    if (loading) return <DetailsSkeleton />
    if (notFound) return <NotFound title="Light novel not found" hint="This light novel isn't in the library any more. It may have been removed." />
    if (error) return <ErrorState error={error} title="Couldn't load this light novel" onRetry={refetch} showBack />
    if (!ln) return null

    const volumes = ln.metadata.volumes ?? [];
    const read = volume => openViewer(navigate, `/light-novels/${id}/volumes/${volume.id}/read`);
    const volumeName = volume => volume.title ?? `Volume ${volume.volume_number}`;
    // the first volume not read yet is where the user is up to
    const nextVolume = volumes.find(v => !readVolumes.has(v.id));
    const nextStarted = nextVolume && (scrollPosition[nextVolume.id] ?? 0) > 0;

    return (
        <div className="flex flex-col gap-8">
            <DetailsHero
                cover={{ src: newCover ?? ln.cover_image_url }}
                eyebrow="Light Novel"
                title={ln.title}
                meta={[
                    ln.metadata.author && `by ${ln.metadata.author}`,
                    ln.release_date && Format.year(ln.release_date),
                    volumes.length > 0 && `${volumes.length} ${volumes.length === 1 ? "volume" : "volumes"}`,
                ]}
                genres={ln.metadata.genres ?? []}
                description={ln.description}
                html
                primary={volumes.length > 0 && <button className="btn btn-primary btn-lg shadow-glow px-6" onClick={() => read(nextVolume ?? volumes[0])}>
                    <BookOpen size={18}/>
                    {nextVolume
                        ? `${nextStarted ? "Continue" : readVolumes.size > 0 ? "Read" : "Start reading"} Vol. ${nextVolume.volume_number}`
                        : "Read again"}
                </button>}
                actions={<>
                    <ListStatusMenu
                        status={userEntry?.status}
                        type={ln.type}
                        options={["current", "completed", "dropped", "planned"]}
                        onSelect={updateStatus}
                        onRemove={removeEntry}
                    />
                    <ActionMenu>
                        <ActionMenuItem icon={ListPlus} label="Add to a collection…" onClick={() => picker.open({ id, type: ln.type, title: ln.title })}/>
                        <ActionMenuItem icon={CheckCheck} label="Mark all read" onClick={() => markAll(true)}/>
                        <ActionMenuItem icon={Undo2} label="Mark all unread" onClick={() => markAll(false)}/>
                        {isAdmin && <>
                            <ActionMenuItem icon={PencilLine} label="Edit details…" onClick={() => editRef.current.showModal()}/>
                            <ActionMenuItem icon={Upload} label="Add volume files…" onClick={() => uploadRef.current.showModal()}/>
                        </>}
                        <DeleteFromServerButton asMenuItem item={{ id, title: ln.title, type: ln.type }} />
                    </ActionMenu>
                </>}
            />

            {volumes.length === 0 ? (
                <EmptyState compact title="No volumes yet" hint="Volumes show up here once they've downloaded."
                    action={isAdmin ? { label: "Add volume files", onClick: () => uploadRef.current.showModal() } : undefined} />
            ) : (
                <ItemList title="Volumes">
                    {volumes.map(volume => {
                        const done = readVolumes.has(volume.id);
                        const pct = scrollPosition[volume.id] ?? 0;
                        const started = !done && pct > 0;
                        return <ItemRow
                            key={volume.id}
                            checked={done}
                            onToggle={() => toggleVolume(volume)}
                            toggleLabel={`Mark ${volumeName(volume)} ${done ? "unread" : "read"}`}
                            number={volume.volume_number}
                            title={volumeName(volume)}
                            progress={started ? pct : null}
                            progressLabel={done ? "Read" : started ? `${Math.round(pct * 100)}%` : null}
                            current={nextVolume?.id === volume.id}
                            onOpen={() => read(volume)}
                            openLabel={`Read ${volumeName(volume)}`}
                            openIcon={BookOpen}
                            menu={pieces.menu({ kind: "volume", id: volume.id, label: volumeName(volume), volume_number: volume.volume_number, title: volume.title })}
                        />
                    })}
                </ItemList>
            )}

            {isAdmin && <>
                <EditDetailsModal dialogRef={editRef} item={ln} html onSaved={({ cover }) => { setNewCover(cover ?? null); refetch(false); }}/>
                <UploadFilesModal
                    dialogRef={uploadRef}
                    kind="volumes"
                    target={{ id, title: ln.title }}
                    have={volumes.map(v => String(v.volume_number))}
                    onAdded={() => refetch(false)}
                />
            </>}
            {pieces.modals}
            {picker.modal}
        </div>
    )
}
