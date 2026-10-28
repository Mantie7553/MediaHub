import { useState, useRef } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { mangaBadge } from "../../utils/status";
import ErrorState from "../../components/states/ErrorState";
import NotFound from "../../components/states/NotFound";
import EmptyState from "../../components/states/EmptyState";
import { DetailsSkeleton } from "../../components/states/Skeletons";
import { toast } from "../../stores/useToastStore";
import { openViewer } from "../../utils/navigation";
import { useMediaItem, useUserContent, useRemoveFromList } from "../../hooks";
import DeleteFromServerButton from "../../components/DeleteFromServerButton";
import api from "../../services/api";
import { BookOpen, CheckCheck, PencilLine, RefreshCw, Undo2, Upload, ListPlus } from "lucide-react";
import DetailsHero from "../../components/details/DetailsHero";
import StatusPill from "../../components/StatusPill";
import { ActionMenu, ActionMenuItem, ListStatusMenu } from "../../components/details/Menus";
import ItemRow, { ItemList } from "../../components/details/ItemRow";
import UploadChaptersModal from "../../components/modals/UploadChaptersModal";
import EditDetailsModal from "../../components/modals/EditDetailsModal";
import useAdminPieces from "../../hooks/useAdminPieces";
import useCollectionPicker from "../../hooks/useCollectionPicker";
import { checkForChapters, saveListEntry } from "../../services/mediaService";
import { userIsAdmin } from "../../utils/auth";

export default function MangaDetailsPage() {
    const navigate = useNavigate();
    const { userContentMap, refresh } = useUserContent();
    const { id } = useParams();
    const { item: manga, loading, error, notFound, refetch } = useMediaItem(id);
    const userEntry = userContentMap[id];
    const removeEntry = useRemoveFromList(userEntry, refresh);
    const [readChapters, setReadChapters] = useState(new Set());
    const [pageProgress, setPageProgress] = useState({});
    const uploadRef = useRef(null);
    const editRef = useRef(null);
    // a cover just uploaded, with a version so the browser doesn't show the one it cached
    const [newCover, setNewCover] = useState(null);
    const isAdmin = userIsAdmin();
    const pieces = useAdminPieces(() => refetch(false));
    const picker = useCollectionPicker();


    // read state and pages start from each copy of the title that loads
    const [shownManga, setShownManga] = useState(null);
    if (manga !== shownManga) {
        setShownManga(manga);
        if (manga?.metadata?.chapters) {
            const initial = {};
            manga.metadata.chapters.forEach(c => { initial[c.id] = c.last_page_read ?? 0; });
            setPageProgress(initial);
            setReadChapters(new Set(manga.metadata.chapters.filter(c => c.completed).map(c => c.id)));
        }
    }

    function updateStatus(status) {
        saveListEntry(userEntry, { media_item_id: id }, { status })
            .then(() => refresh())
            .catch(err => toast.error(err, "Could not update your list"));
    }

    function toggleChapter(chapter) {
        const read = !readChapters.has(chapter.id);
        api.put(`/manga/chapters/${chapter.id}/read`, { read })
            .then(() => {
                const next = new Set(readChapters);
                read ? next.add(chapter.id) : next.delete(chapter.id);
                const status = next.size === manga.metadata.chapters.length ? "completed" : "current";
                setReadChapters(next);
                updateStatus(status);
            })
            .catch(err => toast.error(err, "Could not update the chapter"));
        api.put(`/manga/${id}/chapters/${chapter.id}/progress`,
            { last_page_read: read ? chapter.page_count -1 : 0, completed: read })
        .then(() => setPageProgress(prev => ({ ...prev, [chapter.id]: read ? (chapter.page_count - 1) : 0 })))
        // the read mark above already reports a failure
        .catch(() => {});
    }

    function markAll(read) {
        api.put(`/manga/${id}/read`, { read })
            .then(() => {
                setReadChapters(read ? new Set(manga.metadata.chapters.map(c => c.id)) : new Set());
                updateStatus(read ? "completed" : "planned");
                const newPageProgress = {};
                manga.metadata.chapters.forEach(c => { newPageProgress[c.id] = read ? (c.page_count - 1) : 0; });
                setPageProgress(newPageProgress);
                Promise.all(manga.metadata.chapters.map(c =>
                    api.put(`/manga/${id}/chapters/${c.id}/progress`, { last_page_read: read ? (c.page_count - 1) : 0, completed: read })
                )).catch(err => toast.error(err, "Some chapters' page progress didn't save"));
            })
            .catch(err => toast.error(err, "Could not update the chapters"));
    }

    function handleCheck() {
        checkForChapters(id)
            .then(() => toast.success("Checking MangaDex for chapters you don't have yet. Follow it on the Downloads page"))
            .catch(err => toast.error(err, "Could not start the check"));
    }

    if (loading) return <DetailsSkeleton />
    if (notFound) return <NotFound title="Manga not found" hint="This manga isn't in the library any more. It may have been removed." />
    if (error) return <ErrorState error={error} title="Couldn't load this manga" onRetry={refetch} showBack />
    if (!manga) return null

    const chapters = manga.metadata.chapters ?? [];
    const read = chapter => openViewer(navigate, `/manga/${id}/chapters/${chapter.id}/read`);
    const chapterName = chapter => chapter.title ?? `Chapter ${chapter.chapter_number}`;
    // the first chapter not read yet is where the user is up to
    const nextChapter = chapters.find(c => !readChapters.has(c.id));
    const nextStarted = nextChapter && (pageProgress[nextChapter.id] ?? 0) > 0;

    return (
        <div className="flex flex-col gap-8">
            <DetailsHero
                cover={{ src: newCover ?? manga.cover_image_url, source: manga.external_source }}
                eyebrow="Manga"
                title={manga.title}
                badges={manga.metadata.status && <StatusPill label={manga.metadata.status} tone={mangaBadge(manga.metadata.status)?.replace("badge-", "")}/>}
                meta={[
                    manga.metadata.total_chapters && `${manga.metadata.total_chapters} chapters`,
                    chapters.length > 0 && `${chapters.length} downloaded`,
                ]}
                genres={manga.metadata.genres ?? []}
                description={manga.description}
                primary={chapters.length > 0 && <button className="btn btn-primary btn-lg shadow-glow px-6" onClick={() => read(nextChapter ?? chapters[0])}>
                    <BookOpen size={18}/>
                    {nextChapter
                        ? `${nextStarted ? "Continue" : readChapters.size > 0 ? "Read" : "Start reading"} Ch. ${nextChapter.chapter_number}`
                        : "Read again"}
                </button>}
                actions={<>
                    <ListStatusMenu
                        status={userEntry?.status}
                        type={manga.type}
                        options={["current", "completed", "dropped", "planned"]}
                        onSelect={updateStatus}
                        onRemove={removeEntry}
                    />
                    <ActionMenu>
                        <ActionMenuItem icon={ListPlus} label="Add to a collection…" onClick={() => picker.open({ id, type: manga.type, title: manga.title })}/>
                        <ActionMenuItem icon={CheckCheck} label="Mark all read" onClick={() => markAll(true)}/>
                        <ActionMenuItem icon={Undo2} label="Mark all unread" onClick={() => markAll(false)}/>
                        {isAdmin && <>
                            <ActionMenuItem icon={PencilLine} label="Edit details…" onClick={() => editRef.current.showModal()}/>
                            {manga.external_source === "mangadex" && <ActionMenuItem icon={RefreshCw} label="Check MangaDex for new chapters" onClick={handleCheck}/>}
                            <ActionMenuItem icon={Upload} label="Add chapter files…" onClick={() => uploadRef.current.showModal()}/>
                        </>}
                        <DeleteFromServerButton asMenuItem item={{ id, title: manga.title, type: manga.type }} />
                    </ActionMenu>
                </>}
            />

            {chapters.length === 0 ? (
                <EmptyState compact title="No chapters yet" hint="Chapters show up here once they've downloaded."
                    action={isAdmin ? { label: "Add chapter files", onClick: () => uploadRef.current.showModal() } : undefined} />
            ) : (
                <ItemList title="Chapters">
                    {chapters.map(chapter => {
                        const done = readChapters.has(chapter.id);
                        const { progress, label } = chapterProgress(pageProgress[chapter.id] ?? 0, chapter.page_count, done);
                        return <ItemRow
                            key={chapter.id}
                            checked={done}
                            onToggle={() => toggleChapter(chapter)}
                            toggleLabel={`Mark ${chapterName(chapter)} ${done ? "unread" : "read"}`}
                            number={chapter.chapter_number}
                            title={chapterName(chapter)}
                            progress={progress}
                            progressLabel={label}
                            current={nextChapter?.id === chapter.id}
                            onOpen={() => read(chapter)}
                            openLabel={`Read ${chapterName(chapter)}`}
                            openIcon={BookOpen}
                            menu={pieces.menu({ kind: "chapter", id: chapter.id, label: chapterName(chapter), chapter_number: chapter.chapter_number, title: chapter.title })}
                        />
                    })}
                </ItemList>
            )}

            {isAdmin && <EditDetailsModal dialogRef={editRef} item={manga} onSaved={({ cover }) => { setNewCover(cover ?? null); refetch(false); }}/>}
            {pieces.modals}
            {picker.modal}
            {isAdmin && <UploadChaptersModal
                dialogRef={uploadRef}
                manga={{ id, title: manga.title }}
                haveNumbers={chapters.map(c => c.chapter_number)}
                onUploaded={() => refetch(false)}
            />}
        </div>
    )
}

/**
 * How far into a chapter the user is. Page 0 is saved both for "never opened" and "marked
 * unread" (and for closing the reader on the first page), so a chapter only counts as started
 * past the first page, otherwise it would read 1 / N before it was ever opened
 * @param {number} page the last page read, from 0
 * @param {number} pageCount pages in the chapter
 * @param {boolean} read marked as read
 * @returns progress (0 to 1, null when not started or done) and label
 */
function chapterProgress(page, pageCount, read) {
    if (!pageCount) return { progress: null, label: read ? "Read" : null };
    if (read) return { progress: null, label: "Read" };
    if (page > 0) return { progress: (page + 1) / pageCount, label: `${page + 1} / ${pageCount}` };
    return { progress: null, label: `${pageCount} pages` };
}
