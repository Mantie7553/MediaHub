import { Check, ChevronDown, MoreHorizontal, Plus } from "lucide-react"
import Format from "../../utils/format"
import { mediaStatusDot } from "../../utils/status"

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
    return <div className="dropdown dropdown-end">
        <div tabIndex={0} role="button" aria-label={label}
            className={`btn btn-square border-white/10 bg-white/10 hover:bg-white/15 backdrop-blur ${size === "sm" ? "btn-sm" : ""}`}>
            <MoreHorizontal size={size === "sm" ? 16 : 20}/>
        </div>
        <ul tabIndex={0} className="dropdown-content menu surface-raised z-30 mt-2 w-60 p-1.5 gap-0.5">
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
