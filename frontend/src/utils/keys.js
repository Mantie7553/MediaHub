/**
 * Whether a key press belongs to what has focus rather than to a page's keyboard shortcuts:
 * anything typed into a field, Space or Enter on a button or link (which click it), and keys
 * held with Ctrl, Cmd or Alt (the browser's own shortcuts)
 * @param {KeyboardEvent} e the keydown event
 * @returns true when the page should leave the key alone
 */
export function keyForFocused(e) {
    if (e.ctrlKey || e.metaKey || e.altKey) return true
    const target = e.target
    if (!(target instanceof HTMLElement)) return false
    if (target.isContentEditable || ["INPUT", "TEXTAREA", "SELECT"].includes(target.tagName)) return true
    return (e.key === " " || e.key === "Enter") && ["BUTTON", "A"].includes(target.tagName)
}
