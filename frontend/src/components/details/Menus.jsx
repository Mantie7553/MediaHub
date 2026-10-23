import { useRef, useState } from "react"
import { Check, ChevronDown, MoreHorizontal, Plus } from "lucide-react"
import Format from "../../utils/format"
import { mediaStatusDot } from "../../utils/status"

// sizes for placing an ActionMenu: its width (w-60), an item's height, the list's padding, the
// phone tab bar's height (h-16) and the gap kept from the screen's edges
const MENU_WIDTH_PX = 240
const MENU_ITEM_PX = 40
const MENU_PADDING_PX = 12
const TAB_BAR_PX = 64
const SCREEN_MARGIN_PX = 8

// a daisyUI dropdown stays open while focused, so picking an item closes it by dropping focus
function closeMenu() {
    document.activeElement?.blur()
}

/**
 * The user's list status as a button, with a menu to change it or take the title off the list
 * @param {string} status the user's current status, empty when it isn't on their list
 * @param {string} type the media type, for labels like "Watching" or "Reading"
 * @param {string[]} options the statuses to offer
 * @param {function} onSelect called with the picked status
 * @param {function} onRemove takes it off the list, shown only when it's on the list
 */
export function ListStatusMenu({ status, type, options, onSelect, onRemove }) {
    return <div className="dropdown dropdown-center sm:dropdown-start flex-1 sm:flex-none">
        <div tabIndex={0} role="button" className="btn w-full sm:w-auto border-white/10 bg-white/10 hover:bg-white/15 backdrop-blur gap-2 font-medium">
            {status
                ? <span className={`size-2 rounded-full ${mediaStatusDot(status)}`}></span>
                : <Plus size={16}/>}
            {Format.statusLabel(status, type)}
            <ChevronDown size={16} className="text-base-content/50 ml-auto sm:ml-0"/>
        </div>
        <ul tabIndex={0} className="dropdown-content menu surface-raised z-30 mt-2 w-full min-w-56 p-1.5 gap-0.5">
            {options.map(option => (
                <li key={option}>
                    <button className={option === status ? "text-primary font-medium" : ""} onClick={() => { onSelect(option); closeMenu(); }}>
                        <span className={`size-2 rounded-full ${mediaStatusDot(option)}`}></span>
                        <span className="flex-1">{Format.statusLabel(option, type)}</span>
                        {option === status && <Check size={16}/>}
                    </button>
                </li>
            ))}
            {status && onRemove && (
                <li className="border-t border-white/5 mt-1 pt-1">
                    <button className="text-error" onClick={() => { onRemove(); closeMenu(); }}>
                        Remove from list
                    </button>
                </li>
            )}
        </ul>
    </div>
}

/**
 * A ⋯ button with a menu of less common actions
 * @param {string} label names the menu for screen readers
 * @param {string} size "md", or "sm" inside lists
 * @param {any} children ActionMenuItem elements
 */
export function ActionMenu({ label = "More actions", size = "md", children }) {
    const menuRef = useRef(null)
    const [place, setPlace] = useState({ start: false, top: false })

    // decided as the menu opens: it lines up with the button's right edge unless that runs it off
    // the left of the screen, and opens upwards when there's no room under the button (the phone
    // tab bar and the player bar cover the bottom of the screen)
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

    return <div className={`dropdown ${place.start ? "dropdown-start" : "dropdown-end"} ${place.top ? "dropdown-top" : ""}`}>
        <div tabIndex={0} role="button" aria-label={label} onPointerDown={placeMenu} onFocus={placeMenu}
            className={`btn btn-square border-white/10 bg-white/10 hover:bg-white/15 backdrop-blur ${size === "sm" ? "btn-sm" : ""}`}>
            <MoreHorizontal size={size === "sm" ? 16 : 20}/>
        </div>
        <ul ref={menuRef} tabIndex={0} className={`dropdown-content menu surface-raised z-30 ${place.top ? "mb-2" : "mt-2"} w-60 max-w-[calc(100vw-1rem)] p-1.5 gap-0.5`}>
            {children}
        </ul>
    </div>
}

/**
 * One item in an ActionMenu
 * @param {any} icon a lucide icon component
 * @param {string} label the text
 * @param {function} onClick runs the action, the menu closes afterwards
 * @param {boolean} danger shows it in red
 */
export function ActionMenuItem({ icon: Icon, label, onClick, danger = false }) {
    return <li>
        <button className={danger ? "text-error" : ""} onClick={() => { onClick(); closeMenu(); }}>
            {Icon && <Icon size={16}/>} {label}
        </button>
    </li>
}
