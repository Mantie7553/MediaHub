import { useRef } from "react"
import { ArrowUpDown, Check, ChevronDown, MoreHorizontal, Plus } from "lucide-react"
import Format from "../../utils/format"
import { mediaStatusDot } from "../../utils/status"
import useMenuPlacement from "../../hooks/useMenuPlacement"

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
 * A menu of the ways a list can be sorted, with the picked one ticked. On phones the button is
 * just its icon, to leave the row's heading room
 * @param {object[]} options the list's sort options
 * @param {object} value the picked option
 * @param {function} onChange called with the key of the option picked
 * @param {string} label names the menu for screen readers, like "Sort Anime"
 */
export function SortMenu({ options, value, onChange, label = "Sort" }) {
    const menuRef = useRef(null)
    const [place, placeMenu] = useMenuPlacement(menuRef)
    const sorted = value && value.key !== options[0]?.key

    return <div className={`dropdown ${place.start ? "dropdown-start" : "dropdown-end"} ${place.top ? "dropdown-top" : ""}`}>
        <div tabIndex={0} role="button" aria-label={`${label}: ${value?.label}`} title={value?.label}
            onPointerDown={placeMenu} onFocus={placeMenu}
            className={`flex items-center gap-1.5 h-7 px-2 rounded-full text-sm transition-colors cursor-pointer
                ${sorted ? "bg-primary/15 text-primary" : "bg-white/5 text-base-content/60 hover:text-base-content"}`}>
            <ArrowUpDown size={14}/>
            <span className="hidden sm:inline max-w-40 truncate">{value?.label}</span>
        </div>
        <ul ref={menuRef} tabIndex={0} className={`dropdown-content menu surface-raised z-30 ${place.top ? "mb-2" : "mt-2"} w-60 max-w-[calc(100vw-1rem)] p-1.5 gap-0.5`}>
            {options.map(option => (
                <li key={option.key}>
                    <button className={option.key === value?.key ? "text-primary font-medium" : ""}
                        onClick={() => { onChange(option.key); closeMenu(); }}>
                        <span className="flex-1">{option.label}</span>
                        {option.key === value?.key && <Check size={16}/>}
                    </button>
                </li>
            ))}
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
    const [place, placeMenu] = useMenuPlacement(menuRef)

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
