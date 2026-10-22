import { useState } from "react"
import { Clock, GripVertical, Play } from "lucide-react"
import useAudioStore from "../../stores/useAudioStore"

// 3:07, or --:-- when the length isn't known
function formatDuration(secs) {
    if (!secs) return "--:--"
    const m = Math.floor(secs / 60)
    const s = secs % 60
    return `${m}:${String(s).padStart(2, "0")}`
}

/**
 * The numbered list of tracks on an album or playlist page. Picking one plays the list from there
 * @param {object[]} tracks { key, id (the track's media item id), title, subtitle?, number?,
 *  duration_secs, available? (false greys it out, it has no file to play) }
 * @param {function} onPlay called with the index of the track picked
 * @param {function} menu (track, index) => an ActionMenu for the row, or nothing
 * @param {function} onReorder (from, to) when a row is dragged to another place. Dragging works
 *  with a mouse; on touch screens the row menus move tracks instead
 */
export default function TrackList({ tracks, onPlay, menu, onReorder }) {
    const currentTrack = useAudioStore(state => state.currentTrack)
    const isPlaying = useAudioStore(state => state.isPlaying)
    const [dragging, setDragging] = useState(null)
    const [over, setOver] = useState(null)

    function drop(e, index) {
        e.preventDefault()
        if (dragging !== null && dragging !== index) onReorder(dragging, index)
        setDragging(null)
        setOver(null)
    }

    return <section className="surface p-1.5 sm:p-2 animate-fade-up">
        <div className="flex items-center gap-4 px-3 py-2 text-xs uppercase tracking-wider text-base-content/40 border-b border-white/5 mb-1">
            <span className="w-8 text-right">#</span>
            <span className="flex-1">Title</span>
            <Clock size={14} />
        </div>
        <ol className="flex flex-col gap-0.5">
            {tracks.map((track, i) => {
                const playing = currentTrack?.id === track.id
                const playable = track.available !== false
                // the line shows where the dragged row will land: above rows it moves up past,
                // below rows it moves down past
                const marker = over === i && dragging !== null && dragging !== i
                    ? (dragging > i ? "shadow-[inset_0_2px_0_0_var(--color-primary)]" : "shadow-[inset_0_-2px_0_0_var(--color-primary)]")
                    : ""
                return <li key={track.key ?? track.id}
                    className={`flex items-center gap-1 rounded-field ${marker} ${dragging === i ? "opacity-40" : ""}`}
                    draggable={!!onReorder}
                    onDragStart={onReorder ? e => { setDragging(i); e.dataTransfer.effectAllowed = "move"; e.dataTransfer.setData("text/plain", String(i)) } : undefined}
                    onDragOver={onReorder ? e => { e.preventDefault(); if (over !== i) setOver(i) } : undefined}
                    onDragEnd={onReorder ? () => { setDragging(null); setOver(null) } : undefined}
                    onDrop={onReorder ? e => drop(e, i) : undefined}
                >
                    {onReorder && <span className="hidden pointer-fine:flex text-base-content/25 hover:text-base-content/60 cursor-grab active:cursor-grabbing pl-1" aria-hidden="true">
                        <GripVertical size={16}/>
                    </span>}
                    <button
                        className={`group flex-1 min-w-0 flex items-center gap-4 px-3 py-2.5 rounded-field text-left transition-colors hover:bg-white/5
                            ${playing ? "bg-primary/8" : ""} disabled:opacity-40 disabled:hover:bg-transparent`}
                        onClick={() => onPlay(i)}
                        disabled={!playable}
                        aria-label={playable ? `Play ${track.title}` : `${track.title} hasn't downloaded`}
                    >
                        <span className="w-8 flex justify-end text-sm tabular-nums text-base-content/40">
                            {playing
                                ? <Equalizer animate={isPlaying}/>
                                : <>
                                    <span className="group-hover:hidden">{track.number ?? i + 1}</span>
                                    <Play size={14} fill="currentColor" className="hidden group-hover:block text-primary"/>
                                </>}
                        </span>
                        <span className="flex-1 min-w-0">
                            <span className={`block truncate text-sm font-medium ${playing ? "text-primary" : ""}`}>{track.title}</span>
                            {track.subtitle && <span className="block truncate text-xs text-base-content/50">{track.subtitle}</span>}
                        </span>
                        <span className="text-xs tabular-nums text-base-content/45">{formatDuration(track.duration_secs)}</span>
                    </button>
                    {menu?.(track, i)}
                </li>
            })}
        </ol>
    </section>
}

/**
 * Three bouncing bars marking the track that's playing
 * @param {boolean} animate bounces while playing, still when paused
 */
function Equalizer({ animate }) {
    return <span className="flex items-end gap-[2px] h-3.5 text-primary" aria-label="Now playing">
        {[0, 0.2, 0.4].map(delay => (
            <span key={delay} className={`w-[3px] rounded-sm bg-current ${animate ? "animate-eq" : "h-1.5"}`} style={{ animationDelay: `${delay}s` }}></span>
        ))}
    </span>
}
