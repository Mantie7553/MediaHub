// Every dialog in the app is a daisyUI <dialog class="modal">. These listeners, installed once,
// make all of them close like modals should: a click outside the box, and on phones (where they
// are bottom sheets with a grab handle) a swipe down. Escape is the browser's own.
//
// A dialog in the middle of saving disables its backdrop button; such a dialog stays open, as
// does one with data-locked="true".

// how far a sheet has to be pulled down to close, or how fast a shorter flick has to be
const CLOSE_DISTANCE_PX = 120
const FLICK_DISTANCE_PX = 40
const FLICK_SPEED_PX_PER_MS = 0.6
// how far a finger moves before it counts as a drag either way
const DRAG_START_PX = 8
const SHEET_ANIMATION_MS = 180
// sheets are only sheets below daisyUI's sm breakpoint (sm:modal-middle above it)
const PHONE_QUERY = "(max-width: 639px)"

/** Whether a dialog refuses to close right now */
export function isLocked(dialog) {
    return dialog.dataset.locked === "true" || !!dialog.querySelector(":scope > .modal-backdrop button:disabled")
}

/** The open modal a click landed outside the box of: on the dialog itself or its backdrop */
export function backdropDialog(target) {
    if (!(target instanceof Element)) return null
    if (target.matches("dialog.modal[open]")) return target
    const backdrop = target.closest(".modal-backdrop")
    const dialog = backdrop?.closest("dialog.modal[open]")
    return dialog && backdrop.parentElement === dialog ? dialog : null
}

/**
 * Whether a vertical pull on a sheet should close it
 * @param {number} distance px pulled down
 * @param {number} ms how long the pull took
 */
export function pullCloses(distance, ms) {
    if (distance >= CLOSE_DISTANCE_PX) return true
    return distance >= FLICK_DISTANCE_PX && ms > 0 && distance / ms >= FLICK_SPEED_PX_PER_MS
}

// a touch on something that scrolls, slides or takes text isn't a pull on the sheet
function startsOwnGesture(target, box) {
    if (target.closest('input, textarea, select, [role="slider"], [contenteditable="true"]')) return true
    for (let el = target; el && el !== box.parentElement; el = el.parentElement) {
        if (el.scrollTop > 0) return true
    }
    return false
}

function closeSheet(dialog, box) {
    box.style.transition = `transform ${SHEET_ANIMATION_MS}ms ease-in`
    box.style.transform = "translateY(100%)"
    setTimeout(() => {
        dialog.close()
        box.style.transition = ""
        box.style.transform = ""
    }, SHEET_ANIMATION_MS)
}

function snapBack(box) {
    box.style.transition = `transform ${SHEET_ANIMATION_MS}ms ease-out`
    box.style.transform = ""
    setTimeout(() => { box.style.transition = "" }, SHEET_ANIMATION_MS)
}

/** Install the listeners; returns a function that removes them */
export function installModalGestures(doc = document, win = window) {
    // a click only closes when it also started outside the box, so selecting text in a field
    // and letting go outside doesn't close the dialog
    let pressedOn = null
    function onPointerDown(e) {
        pressedOn = backdropDialog(e.target)
    }
    function onClick(e) {
        const dialog = backdropDialog(e.target)
        const started = pressedOn
        pressedOn = null
        if (!dialog) return
        // capture phase, before React: a dialog inside a card's link mustn't follow the link
        e.preventDefault()
        e.stopPropagation()
        // detail 0: a keyboard press on the backdrop's button, which has no pointer to start
        if ((dialog === started || e.detail === 0) && !isLocked(dialog)) dialog.close()
    }

    let pull = null
    function onTouchStart(e) {
        pull = null
        if (e.touches.length !== 1 || !win.matchMedia(PHONE_QUERY).matches) return
        const box = e.target.closest?.("dialog.modal-bottom[open] > .modal-box")
        if (!box || isLocked(box.parentElement) || startsOwnGesture(e.target, box)) return
        const t = e.touches[0]
        pull = { box, dialog: box.parentElement, x: t.clientX, y: t.clientY, at: e.timeStamp, dragging: false, dy: 0 }
    }
    function onTouchMove(e) {
        if (!pull) return
        const t = e.touches[0]
        const dx = t.clientX - pull.x
        const dy = t.clientY - pull.y
        if (!pull.dragging) {
            if (dy > DRAG_START_PX && dy > Math.abs(dx)) {
                pull.dragging = true
                pull.box.style.transition = "none"
            } else if (Math.abs(dx) > DRAG_START_PX || dy < -DRAG_START_PX) {
                pull = null
                return
            } else {
                return
            }
        }
        e.preventDefault()
        pull.dy = Math.max(0, dy)
        pull.box.style.transform = `translateY(${pull.dy}px)`
    }
    function onTouchEnd(e) {
        if (!pull) return
        const { box, dialog, dragging, dy, at } = pull
        pull = null
        if (!dragging) return
        if (pullCloses(dy, e.timeStamp - at)) closeSheet(dialog, box)
        else snapBack(box)
    }

    doc.addEventListener("pointerdown", onPointerDown, true)
    doc.addEventListener("click", onClick, true)
    doc.addEventListener("touchstart", onTouchStart, { capture: true, passive: true })
    doc.addEventListener("touchmove", onTouchMove, { capture: true, passive: false })
    doc.addEventListener("touchend", onTouchEnd, true)
    doc.addEventListener("touchcancel", onTouchEnd, true)
    return () => {
        doc.removeEventListener("pointerdown", onPointerDown, true)
        doc.removeEventListener("click", onClick, true)
        doc.removeEventListener("touchstart", onTouchStart, { capture: true })
        doc.removeEventListener("touchmove", onTouchMove, { capture: true })
        doc.removeEventListener("touchend", onTouchEnd, true)
        doc.removeEventListener("touchcancel", onTouchEnd, true)
    }
}
