import { useCallback, useEffect, useLayoutEffect, useRef } from "react"
import { clampValue, itemLook, scrollForValue, valueAtScroll, wheelLabel } from "../utils/wheel"

// each number's slot, in px; the wheel scrolls in steps of this
const ITEM_PX = 44
// how long scrolling has to stop before the number in the middle is picked
const SETTLE_MS = 120
// a mouse drag shorter than this is a click on a number
const DRAG_PX = 4
// a vertical mouse wheel moves one number per notch, no faster than this
const WHEEL_STEP_MS = 60

// whether the wheel is on screen: not inside a closed dialog
function isShown(el) {
    const dialog = el.closest("dialog")
    return el.clientWidth > 0 && (!dialog || dialog.open)
}

/**
 * A horizontal wheel of numbers, like a phone's time picker turned on its side: swipe, drag,
 * scroll, click a number or use the arrow keys, and the number that stops in the middle is
 * picked. Numbers shrink and fade towards the edges
 * @param {number} value the number picked
 * @param {function} onChange called with the new number once the wheel settles
 * @param {number} min the lowest number
 * @param {number} max the highest number
 * @param {number} unsetValue the number meaning "not set", shown as unsetLabel
 * @param {string} unsetLabel what the unset number shows
 * @param {string} label names the wheel for screen readers
 */
export default function WheelPicker({ value, onChange, min = 0, max = 10, unsetValue = null, unsetLabel = "–", label }) {
    const ref = useRef(null)
    const settleTimer = useRef(null)
    const frame = useRef(0)
    // a mouse drag in progress: where it started and whether it moved far enough to count
    const drag = useRef(null)
    // a click right after a drag is the end of the drag, not a pick
    const justDragged = useRef(false)
    // the first placement jumps straight to the value, later ones glide
    const placed = useRef(false)
    // the value to place the wheel on when it appears
    const latest = useRef(value)
    // the value the wheel is scrolling to by itself (to show a value it was given), null while
    // the user moves it. Only the user's scrolling picks a value: a dialog opening scrolls the
    // wheel too, from wherever it was left last time
    const syncing = useRef(null)
    const count = Math.max(0, max - min + 1)
    const current = clampValue(value, min, max)

    // shrink and fade the numbers around the middle; only the ones in view are touched
    const paint = useCallback(() => {
        frame.current = 0
        const el = ref.current
        if (!el) return
        const centre = el.scrollLeft / ITEM_PX
        const reach = Math.ceil(el.clientWidth / ITEM_PX / 2) + 1
        const from = Math.max(0, Math.floor(centre) - reach)
        const to = Math.min(el.children.length - 1, Math.ceil(centre) + reach)
        for (let i = from; i <= to; i++) {
            const { scale, opacity } = itemLook(i - centre)
            el.children[i].style.transform = `scale(${scale})`
            el.children[i].style.opacity = opacity
        }
    }, [])

    const schedulePaint = useCallback(() => {
        if (!frame.current) frame.current = requestAnimationFrame(paint)
    }, [paint])

    // own: the wheel is showing a value it was given rather than one the user is picking
    const scrollTo = useCallback((v, smooth, own = false) => {
        const el = ref.current
        if (!el) return
        const left = scrollForValue(v, ITEM_PX, min, max)
        if (own) syncing.current = Math.abs(el.scrollLeft - left) >= 1 ? clampValue(v, min, max) : null
        el.scrollTo({ left, behavior: smooth ? "smooth" : "instant" })
        schedulePaint()
    }, [min, max, schedulePaint])

    // the user has hold of the wheel: what it settles on is theirs
    function userTookOver() {
        syncing.current = null
    }

    // the value changed from outside (a new entry, the arrow keys): put it in the middle
    useLayoutEffect(() => {
        latest.current = current
        const el = ref.current
        if (!el || drag.current) return
        if (valueAtScroll(el.scrollLeft, ITEM_PX, min, max) !== current || !placed.current) {
            // a wheel in a closed dialog jumps: a glide started while hidden is cut short when
            // the dialog opens
            scrollTo(current, placed.current && isShown(el), true)
            placed.current = true
        }
        schedulePaint()
    }, [current, min, max, scrollTo, schedulePaint])

    useEffect(() => () => {
        clearTimeout(settleTimer.current)
        cancelAnimationFrame(frame.current)
    }, [])

    // placed again whenever its dialog opens or it changes width: a closed dialog may have no size
    // (display: none) or be laid out but hidden, and opening one can lose a scroll made before
    useEffect(() => {
        const el = ref.current
        if (!el) return
        const place = () => {
            if (el.clientWidth > 0 && !drag.current) scrollTo(latest.current, false, true)
        }
        const resize = typeof ResizeObserver === "undefined" ? null : new ResizeObserver(place)
        resize?.observe(el)
        const dialog = el.closest("dialog")
        const opened = dialog && new MutationObserver(() => { if (dialog.open) requestAnimationFrame(place) })
        opened?.observe(dialog, { attributes: true, attributeFilter: ["open"] })
        return () => { resize?.disconnect(); opened?.disconnect() }
    }, [scrollTo])

    function settle() {
        clearTimeout(settleTimer.current)
        const el = ref.current
        if (!el || drag.current) return
        const v = valueAtScroll(el.scrollLeft, ITEM_PX, min, max)
        if (syncing.current !== null) {
            // still on the way to the value it was given; arriving isn't a pick either
            if (v === syncing.current) syncing.current = null
            return
        }
        if (v !== current) onChange(v)
    }

    function handleScroll() {
        schedulePaint()
        clearTimeout(settleTimer.current)
        settleTimer.current = setTimeout(settle, SETTLE_MS)
    }

    // a vertical mouse wheel steps one number at a time; sideways scrolling (trackpads) is native.
    // Added by hand because React's wheel listener is passive and can't stop the page scrolling
    useEffect(() => {
        const el = ref.current
        if (!el) return
        let last = 0
        function onWheel(e) {
            syncing.current = null
            if (Math.abs(e.deltaY) <= Math.abs(e.deltaX)) return
            e.preventDefault()
            const now = performance.now()
            if (now - last < WHEEL_STEP_MS) return
            last = now
            const from = valueAtScroll(el.scrollLeft, ITEM_PX, min, max)
            scrollTo(from + Math.sign(e.deltaY), true)
        }
        el.addEventListener("wheel", onWheel, { passive: false })
        return () => el.removeEventListener("wheel", onWheel)
    }, [min, max, scrollTo])

    // touch scrolls natively; a mouse has to be dragged by hand
    function onPointerDown(e) {
        userTookOver()
        if (e.pointerType !== "mouse" || e.button !== 0) return
        drag.current = { id: e.pointerId, x: e.clientX, left: ref.current.scrollLeft, moved: false }
    }

    function onPointerMove(e) {
        const d = drag.current
        if (!d) return
        const el = ref.current
        const dx = e.clientX - d.x
        if (!d.moved && Math.abs(dx) > DRAG_PX) {
            d.moved = true
            // snapping would fight every step of the drag
            el.style.scrollSnapType = "none"
            el.setPointerCapture(d.id)
        }
        if (d.moved) el.scrollLeft = d.left - dx
    }

    function onPointerUp() {
        const d = drag.current
        if (!d) return
        const el = ref.current
        drag.current = null
        if (!d.moved) return
        el.releasePointerCapture?.(d.id)
        el.style.scrollSnapType = ""
        justDragged.current = true
        setTimeout(() => { justDragged.current = false }, 0)
        // picked right away: a drag that stops exactly on a number (always so at either end)
        // leaves nothing to scroll, so the wheel would never settle and pick it
        const v = valueAtScroll(el.scrollLeft, ITEM_PX, min, max)
        scrollTo(v, true)
        if (v !== current) onChange(v)
    }

    function onKeyDown(e) {
        const step = { ArrowLeft: -1, ArrowDown: -1, ArrowRight: 1, ArrowUp: 1, PageDown: -5, PageUp: 5 }[e.key]
        let next
        if (step) next = current + step
        else if (e.key === "Home") next = min
        else if (e.key === "End") next = max
        else return
        e.preventDefault()
        next = clampValue(next, min, max)
        if (next !== current) onChange(next)
    }

    return <div className="relative w-full max-w-xs">
        {/* the middle slot */}
        <div className="pointer-events-none absolute inset-y-1 left-1/2 -translate-x-1/2 rounded-field bg-primary/15 ring-1 ring-primary/40"
            style={{ width: ITEM_PX }} aria-hidden="true"></div>
        <div
            ref={ref}
            role="slider"
            tabIndex={0}
            aria-label={label}
            aria-valuemin={min}
            aria-valuemax={max}
            aria-valuenow={current}
            aria-valuetext={wheelLabel(current, unsetValue, unsetLabel)}
            onScroll={handleScroll}
            onScrollEnd={settle}
            onKeyDown={onKeyDown}
            onPointerDown={onPointerDown}
            onTouchStart={userTookOver}
            onPointerMove={onPointerMove}
            onPointerUp={onPointerUp}
            onPointerCancel={onPointerUp}
            className="relative flex h-12 overflow-x-auto overscroll-x-contain snap-x snap-mandatory select-none rounded-field
                cursor-grab [scrollbar-width:none] [&::-webkit-scrollbar]:hidden
                [mask-image:linear-gradient(to_right,transparent,#000_30%,#000_70%,transparent)]"
            style={{ paddingInline: `calc(50% - ${ITEM_PX / 2}px)` }}
        >
            {Array.from({ length: count }, (_, i) => {
                const v = min + i
                return <button
                    key={v}
                    type="button"
                    tabIndex={-1}
                    aria-hidden="true"
                    onClick={() => { if (!justDragged.current) scrollTo(v, true) }}
                    className={`shrink-0 snap-center grid place-items-center text-lg font-semibold tabular-nums will-change-transform
                        transition-colors ${v === current ? "text-primary" : "text-base-content"}`}
                    style={{ width: ITEM_PX }}
                >
                    {wheelLabel(v, unsetValue, unsetLabel)}
                </button>
            })}
        </div>
    </div>
}
