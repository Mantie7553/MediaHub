import { useRef, useState } from "react"
import Format from "../../utils/format"
import { seekFraction, segmentMarks } from "../../utils/player"

/**
 * The video's progress bar: what's loaded, what's played, the parts that can be skipped, and
 * dragging or clicking it to seek. The arrow keys seek through the player's own shortcuts
 * @param {number} currentTime seconds played
 * @param {number} duration the video's length, 0 while unknown
 * @param {number} buffered seconds loaded up to
 * @param {object[]} segments the skippable parts, [{ type, start, end }]
 * @param {function} onSeek called with the seconds to go to
 * @param {function} onScrubbing called with true while it's being dragged, false after
 */
export default function SeekBar({ currentTime, duration, buffered, segments, onSeek, onScrubbing }) {
    const barRef = useRef(null)
    // 0 to 1 while dragging, where the video goes on release
    const [drag, setDrag] = useState(null)
    // where a mouse is over the bar, for the time tooltip
    const [hover, setHover] = useState(null)

    const known = duration > 0
    const played = drag ?? (known ? currentTime / duration : 0)
    const loaded = known ? Math.min(buffered / duration, 1) : 0
    const tooltip = drag ?? hover

    function fraction(e) {
        return seekFraction(e.clientX, barRef.current.getBoundingClientRect())
    }

    function onPointerDown(e) {
        if (!known || e.button !== 0) return
        e.currentTarget.setPointerCapture(e.pointerId)
        setDrag(fraction(e))
        onScrubbing(true)
    }
    function onPointerMove(e) {
        if (drag !== null) setDrag(fraction(e))
        if (e.pointerType === "mouse" && known) setHover(fraction(e))
    }
    function onPointerUp(e) {
        if (drag === null) return
        onSeek(fraction(e) * duration)
        setDrag(null)
        onScrubbing(false)
    }
    function onPointerCancel() {
        if (drag === null) return
        setDrag(null)
        onScrubbing(false)
    }

    return <div
        ref={barRef}
        className="group/seek relative h-6 flex items-center cursor-pointer touch-none select-none"
        role="slider"
        tabIndex={0}
        aria-label="Seek"
        aria-valuemin={0}
        aria-valuemax={Math.floor(duration)}
        aria-valuenow={Math.floor(played * duration)}
        aria-valuetext={`${Format.duration(played * duration)} of ${Format.duration(duration)}`}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUp}
        onPointerCancel={onPointerCancel}
        onPointerLeave={() => setHover(null)}
    >
        <div className={`relative w-full rounded-full bg-white/20 transition-[height] duration-150 ${drag !== null ? "h-1.5" : "h-1 group-hover/seek:h-1.5"}`}>
            <div className="absolute inset-y-0 left-0 rounded-full bg-white/30" style={{ width: `${loaded * 100}%` }}></div>
            {segmentMarks(segments, duration).map(mark => (
                <div key={`${mark.type}-${mark.left}`} className="absolute inset-y-0 bg-white/40" style={{ left: `${mark.left}%`, width: `${mark.width}%` }}></div>
            ))}
            <div className="absolute inset-y-0 left-0 rounded-full bg-primary shadow-[0_0_8px_rgb(242_89_18/0.8)]" style={{ width: `${played * 100}%` }}></div>
            <div
                className={`absolute top-1/2 size-3.5 -translate-x-1/2 -translate-y-1/2 rounded-full bg-primary shadow transition-transform duration-150 ${drag !== null ? "scale-110" : "scale-0 group-hover/seek:scale-100 group-focus-visible/seek:scale-100"}`}
                style={{ left: `${played * 100}%` }}
            ></div>
        </div>
        {tooltip !== null && known && (
            <span
                className="absolute bottom-full mb-2 -translate-x-1/2 rounded-md bg-black/80 px-2 py-0.5 text-xs font-medium tabular-nums pointer-events-none"
                style={{ left: `clamp(1.5rem, ${tooltip * 100}%, calc(100% - 1.5rem))` }}
            >
                {Format.duration(tooltip * duration)}
            </span>
        )}
    </div>
}
