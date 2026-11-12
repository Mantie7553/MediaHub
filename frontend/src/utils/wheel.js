// the maths behind components/WheelPicker.jsx, kept apart so it can be tested

/** Keep a value within min and max */
export function clampValue(value, min, max) {
    return Math.min(max, Math.max(min, Math.round(Number(value) || 0)))
}

/** The value sitting in the middle of the wheel when it is scrolled this far */
export function valueAtScroll(scrollLeft, itemWidth, min, max) {
    return clampValue(min + Math.round(scrollLeft / itemWidth), min, max)
}

/** How far to scroll the wheel to put a value in the middle */
export function scrollForValue(value, itemWidth, min, max) {
    return (clampValue(value, min, max) - min) * itemWidth
}

/** The text shown for a value, the unset label for the unset value */
export function wheelLabel(value, unsetValue, unsetLabel) {
    return value === unsetValue ? unsetLabel : String(value)
}

/**
 * How an item looks by how far it is from the middle, in items: shrinking and fading out
 * @returns { scale, opacity }
 */
export function itemLook(distance) {
    const d = Math.min(Math.abs(distance), 3)
    return { scale: 1 - d * 0.14, opacity: 1 - d * 0.27 }
}
