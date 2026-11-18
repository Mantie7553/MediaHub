import { useState } from "react"

// sizes for placing a dropdown menu: its width (w-60), an item's height, the list's padding, the
// phone tab bar's height (h-16) and the gap kept from the screen's edges
const MENU_WIDTH_PX = 240
const MENU_ITEM_PX = 40
const MENU_PADDING_PX = 12
const TAB_BAR_PX = 64
const SCREEN_MARGIN_PX = 8

/**
 * Where a dropdown menu opens, decided as it opens: it lines up with the button's right edge
 * unless that runs it off the left of the screen, and opens upwards when there's no room under
 * the button (the phone tab bar and the player bar cover the bottom of the screen)
 * @param {object} menuRef the menu's list, whose items give its height while it's still hidden
 * @returns [place, placeMenu]: { start, top }, and a handler for the button's pointer down and focus
 */
export default function useMenuPlacement(menuRef) {
    const [place, setPlace] = useState({ start: false, top: false })

    function placeMenu(e) {
        const rect = e.currentTarget.getBoundingClientRect()
        const root = document.documentElement
        const covered = (parseFloat(getComputedStyle(root).getPropertyValue("--player-h")) || 0) +
            (window.matchMedia("(min-width: 1024px)").matches ? 0 : TAB_BAR_PX)
        // the menu is hidden until it opens, so its height comes from its items
        const height = (menuRef.current?.children.length ?? 0) * MENU_ITEM_PX + MENU_PADDING_PX
        const below = window.innerHeight - covered - rect.bottom
        setPlace({
            start: rect.right - MENU_WIDTH_PX < SCREEN_MARGIN_PX,
            top: below < height + SCREEN_MARGIN_PX && rect.top > below,
        })
    }

    return [place, placeMenu]
}
