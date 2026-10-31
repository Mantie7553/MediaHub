import { ChevronDown, ChevronUp } from 'lucide-react'
import { useState, useEffect, useLayoutEffect, useMemo, useRef, useCallback } from "react"
import { useParams, useLocation, useNavigate } from "react-router-dom"
import api from "../../services/api"
import { getCached, setCached } from "../../services/apiCache"
import { useMediaItem } from "../../hooks"
import useApi from "../../hooks/useApi"
import ErrorState from "../../components/states/ErrorState"
import NotFound from "../../components/states/NotFound"
import { ReaderSkeleton, TextSkeleton } from "../../components/states/Skeletons"
import { exitViewer, switchViewer } from "../../utils/navigation"
import ReaderToolbar from "../../components/ReaderToolbar"
import useDocumentTitle from "../../hooks/useDocumentTitle"
import { sanitizeHtml } from "../../utils/sanitize"
import {
    BLOCK_SELECTOR, anchorAtTop, blockBelow, bookmarkBlockForTap, getBlocks, savedPositionKey, scrollToAnchor,
} from "../../utils/readerAnchor"

// how long opening waits for the images above the reading position before showing the page
const IMAGE_WAIT_MS = 1500
// how long the position is held in place against late images and fonts if the reader doesn't move
const PIN_MS = 5000
// room left above the bookmark line when a volume opens on it
const BOOKMARK_MARGIN = 24

export default function LightNovelReader() {
    const { id, volumeId } = useParams()
    const { item: ln, loading, error, notFound, refetch } = useMediaItem(id)

    if (loading) return <ReaderSkeleton variant="text" />
    if (notFound) return <NotFound title="Light novel not found" hint="This light novel isn't in the library any more." />
    if (error) return <ErrorState error={error} title="Couldn't open this light novel" onRetry={refetch} showBack />
    if (!ln) return null

    const volumes = ln.metadata?.volumes ?? [];
    const volume = volumes.find(v => v.id === volumeId);
    if (!volume) return <NotFound title="Volume not found" hint="This volume isn't part of the light novel any more. Pick another from the volume list." />
    return <LightNovelReaderInner key={volumeId} id={id} volumeId={volumeId} volume={volume} volumes={volumes} title={ln.title} />
}

function LightNovelReaderInner({ id, volumeId, volume, volumes, title }) {
    useDocumentTitle(`${title} · Vol. ${volume.volume_number}`);
    const navigate = useNavigate();
    const location = useLocation();
    // where to open: the last save in this tab beats the title's data, which can be from before it
    const [start] = useState(() => getCached(savedPositionKey(volumeId)) ?? {
        pct: volume.scroll_position ?? 0,
        anchor: volume.anchor_block != null ? { block: volume.anchor_block, char: volume.anchor_char ?? 0 } : null,
        bookmark: volume.bookmark_block ?? null,
    });
    const hasStart = start.bookmark != null || start.anchor != null || start.pct > 0;

    const { data: content, loading, error, reload } = useApi(`/light-novels/${id}/volumes/${volumeId}/content`);
    // an EPUB is a file from anywhere: its markup can't be allowed to run code in the page. React
    // rewrites the markup whenever this object is a new one, which would swap out every element a
    // position or bookmark points at on each scroll, so it's made once per volume
    const html = useMemo(() => ({ __html: sanitizeHtml(content) }), [content]);
    const [scrollPct, setScrollPct] = useState(start.pct);
    // the page stays hidden until it's scrolled to where the reader was, so it doesn't jump about
    const [restored, setRestored] = useState(!hasStart);
    const contentRef = useRef(null);
    const articleRef = useRef(null);
    const pageRef = useRef(null);
    const lineRef = useRef(null);
    const blocksRef = useRef([]);
    const scrollPctRef = useRef(start.pct);
    const anchorRef = useRef(start.bookmark != null ? { block: start.bookmark, char: 0 } : start.anchor);
    const bookmarkRef = useRef(start.bookmark);
    // while set, scrolling is the restore placing the page, not the reader moving
    const restoringRef = useRef(hasStart);
    const debounceTimer = useRef(null);
    const anchorTimer = useRef(null);
    const currentIndex = volumes.findIndex(v => v.id === volumeId);
    const nextVolume = volumes[currentIndex + 1] ?? null;
    const prevVolume = volumes[currentIndex - 1] ?? null;

    const remember = useCallback(() => {
        setCached(savedPositionKey(volumeId), {
            pct: scrollPctRef.current, anchor: anchorRef.current, bookmark: bookmarkRef.current,
        })
    }, [volumeId])

    const saveProgress = useCallback(() => {
        const pct = scrollPctRef.current
        const anchor = anchorRef.current
        api.put(`/light-novels/volumes/${volumeId}/progress`, {
            scroll_position: pct,
            ...(anchor && { anchor_block: anchor.block, anchor_char: anchor.char }),
        }).catch(() => {})
        remember()
        if (pct >= 1) {
            // best effort like the progress save above, reading carries on either way
            api.put(`/light-novels/volumes/${volumeId}/read`, { read: true }).catch(() => {})
        }
    }, [volumeId, remember])

    // draw the bookmark line in the gap above its block, or hide it
    const placeLine = useCallback(() => {
        const line = lineRef.current
        const page = pageRef.current
        if (!line || !page) return
        const block = blocksRef.current[bookmarkRef.current]
        line.style.display = block ? "" : "none"
        if (block) line.style.top = `${block.getBoundingClientRect().top - page.getBoundingClientRect().top - 9}px`
    }, [])

    const setBookmark = useCallback(block => {
        bookmarkRef.current = block
        placeLine()
        api.put(`/light-novels/volumes/${volumeId}/bookmark`, { block }).catch(() => {})
        remember()
    }, [volumeId, placeLine, remember])

    // find the blocks, then put the page where the reader was. Images above that spot load after
    // the text and push it down, so it waits for them and holds the spot until the reader moves
    useLayoutEffect(() => {
        const el = contentRef.current
        const article = articleRef.current
        if (!el || !article) return
        const blocks = getBlocks(article)
        blocksRef.current = blocks
        placeLine()

        let pinned = restoringRef.current
        const target = start.bookmark != null ? { block: start.bookmark, char: 0 } : start.anchor
        const margin = start.bookmark != null ? BOOKMARK_MARGIN : 0
        function place() {
            if (target && scrollToAnchor(el, blocks, target, margin)) return
            el.scrollTop = start.pct * (el.scrollHeight - el.clientHeight)
        }

        const observer = new ResizeObserver(() => {
            placeLine()
            if (pinned) place()
        })
        observer.observe(article)
        if (!pinned) return () => observer.disconnect()

        function release() {
            pinned = false
            restoringRef.current = false
            unlisten()
        }
        function unlisten() {
            el.removeEventListener("wheel", release)
            el.removeEventListener("touchstart", release)
            window.removeEventListener("pointerdown", release)
            window.removeEventListener("keydown", release)
        }

        // only the images above the spot move it; an old save with just a percentage needs them all
        const targetBlock = target && blocks[target.block]
        const waiting = Array.from(article.querySelectorAll("img")).filter(img => !img.complete && (!targetBlock ||
            (img.compareDocumentPosition(targetBlock) & Node.DOCUMENT_POSITION_FOLLOWING)))
        const loaded = Promise.all(waiting.map(img => new Promise(resolve => {
            img.addEventListener("load", resolve, { once: true })
            img.addEventListener("error", resolve, { once: true })
        })))
        let waitTimer
        let pinTimer
        let cancelled = false
        Promise.race([loaded, new Promise(resolve => { waitTimer = setTimeout(resolve, IMAGE_WAIT_MS) })]).then(() => {
            if (cancelled) return
            place()
            setRestored(true)
            el.addEventListener("wheel", release, { passive: true })
            el.addEventListener("touchstart", release, { passive: true })
            window.addEventListener("pointerdown", release)
            window.addEventListener("keydown", release)
            pinTimer = setTimeout(release, PIN_MS)
        })

        return () => {
            cancelled = true
            pinned = false
            observer.disconnect()
            clearTimeout(waitTimer)
            clearTimeout(pinTimer)
            unlisten()
        }
    }, [loading, start, placeLine])

    // track scroll position
    useEffect(() => {
        const el = contentRef.current
        if (!el) return

        function handleScroll() {
            const maxScroll = el.scrollHeight - el.clientHeight
            const pct = maxScroll > 0 ? el.scrollTop / maxScroll : 0
            scrollPctRef.current = pct
            setScrollPct(pct)
            if (restoringRef.current) return

            // a screen read past the bookmark and it's done with, reopening goes by the position again
            const mark = blocksRef.current[bookmarkRef.current]
            if (mark && mark.getBoundingClientRect().top < el.getBoundingClientRect().top - el.clientHeight) {
                setBookmark(null)
            }

            clearTimeout(anchorTimer.current)
            anchorTimer.current = setTimeout(() => {
                anchorRef.current = anchorAtTop(el, articleRef.current, blocksRef.current) ?? anchorRef.current
            }, 150)
            clearTimeout(debounceTimer.current)
            debounceTimer.current = setTimeout(saveProgress, 2000)
        }

        el.addEventListener("scroll", handleScroll)
        return () => {
            el.removeEventListener("scroll", handleScroll)
            clearTimeout(debounceTimer.current)
            clearTimeout(anchorTimer.current)
        }
    }, [loading, saveProgress, setBookmark])

    // save on unmount
    useEffect(() => {
        return () => {
            saveProgress()
        }
    }, [saveProgress])

    // a tap on the text puts the bookmark between the paragraphs nearest it, a tap on it takes it off
    function handleTap(e) {
        if (!window.getSelection()?.isCollapsed || e.target.closest("a")) return
        const blocks = blocksRef.current
        const block = e.target.closest(BLOCK_SELECTOR)
        const tapped = block ? blocks.indexOf(block) : -1
        let index
        if (tapped >= 0) {
            const rect = block.getBoundingClientRect()
            index = bookmarkBlockForTap(rect.top, rect.height, e.clientY, tapped, blocks.length)
        } else {
            index = blockBelow(blocks, e.clientY)
            if (index < 0) return
        }
        setBookmark(index === bookmarkRef.current ? null : index)
    }

    if (error) return <ErrorState error={error} title="Couldn't load this volume" onRetry={reload} showBack />

    const atStart = scrollPct <= 0
    const atEnd = scrollPct >= 1

    return (
        <div className="flex flex-col h-dvh">
            <ReaderToolbar
                onBack={() => exitViewer(navigate, location, `/light-novels/${id}`)}
                backLabel="Volumes"
                title={title}
                subtitle={volume.title ?? `Volume ${volume.volume_number}`}
                progress={scrollPct}
                progressLabel={`${Math.round(scrollPct * 100)}%`}
            >
                <button
                    className="btn btn-ghost btn-sm gap-1"
                    disabled={atStart && !prevVolume}
                    aria-label={atStart && prevVolume ? "Previous volume" : "Back to top"}
                    onClick={() => {
                        if (atStart && prevVolume) {
                            saveProgress();
                            switchViewer(navigate, location, `/light-novels/${id}/volumes/${prevVolume.id}/read`);
                        } else {
                            contentRef.current?.scrollTo({ top: 0, behavior: 'smooth' });
                        }
                    }}>
                    <ChevronUp size={18}/>
                    <span className={atStart && prevVolume ? "" : "hidden md:inline"}>{atStart && prevVolume ? "Prev volume" : "Top"}</span>
                </button>
                <button
                    className={`btn btn-sm gap-1 ${atEnd && nextVolume ? "btn-primary" : "btn-ghost"}`}
                    disabled={atEnd && !nextVolume}
                    aria-label={atEnd && nextVolume ? "Next volume" : "Jump to the end"}
                    onClick={() => {
                        if (atEnd && nextVolume) {
                            saveProgress();
                            switchViewer(navigate, location, `/light-novels/${id}/volumes/${nextVolume.id}/read`);
                        } else {
                            const el = contentRef.current;
                            if (el) el.scrollTo({ top: el.scrollHeight, behavior: 'smooth' });
                        }
                    }}>
                    <ChevronDown size={18}/>
                    <span className={atEnd && nextVolume ? "" : "hidden md:inline"}>{atEnd && nextVolume ? "Next volume" : "End"}</span>
                </button>
            </ReaderToolbar>
            {/* the toolbar stays up while the text loads */}
            {loading ? <TextSkeleton /> : <div
                ref={contentRef}
                className="flex-1 overflow-y-auto flex flex-col items-center px-3 sm:px-6 py-4 sm:py-8 pb-[calc(2rem+var(--player-h,0px)+env(safe-area-inset-bottom))]"
            >
                {/* the bookmark line sits beside the volume's markup, not in it, so a render never wipes it */}
                <div ref={pageRef} className={`relative w-full max-w-2xl ${restored ? "" : "invisible"}`}>
                    {/* a warm page to read on, the text comes with its own markup from the EPUB */}
                    <article
                        ref={articleRef}
                        onClick={handleTap}
                        className="w-full rounded-box bg-[#f7f2e8] text-[#1f1a14] shadow-raised px-5 sm:px-12 py-8 sm:py-12
                            text-[1.0625rem] leading-relaxed font-[Georgia,'Times_New_Roman',serif]
                            [&_p]:mb-4 [&_img]:max-w-full [&_img]:h-auto [&_img]:mx-auto [&_img]:my-6 [&_img]:rounded
                            [&_h1]:text-2xl [&_h1]:font-semibold [&_h1]:mb-4 [&_h2]:text-xl [&_h2]:font-semibold [&_h2]:mb-3 [&_h3]:font-semibold [&_h3]:mb-2"
                        dangerouslySetInnerHTML={html}
                    />
                    <div ref={lineRef} aria-hidden="true"
                        className="pointer-events-none absolute left-5 right-5 sm:left-12 sm:right-12 h-0.5 rounded-full bg-orange-500" />
                </div>
            </div>}
        </div>
    )
}
