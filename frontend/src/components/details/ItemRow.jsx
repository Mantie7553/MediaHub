import { Check, Play } from "lucide-react"

/**
 * One episode, chapter or volume in a details page list: a watched/read toggle, its number and
 * title with progress under it, and a button to open it. The whole row opens it on touch screens
 * @param {boolean} checked watched or read
 * @param {function} onToggle flips checked
 * @param {string} toggleLabel e.g. "Mark episode 3 watched"
 * @param {any} number shown in the left column, like 03
 * @param {string} title the name
 * @param {number|null} progress 0 to 1 for a bar under the title, null for none
 * @param {string} progressLabel beside the bar, like "12 min left"
 * @param {boolean} current highlights the row the user is up to
 * @param {function} onOpen plays or reads it
 * @param {string} openLabel e.g. "Watch episode 3"
 * @param {any} openIcon a lucide icon, Play by default
 */
export default function ItemRow({ checked, onToggle, toggleLabel, number, title, progress = null, progressLabel, current = false, onOpen, openLabel, openIcon: OpenIcon = Play }) {
    return <li className={`group relative flex items-center gap-3 sm:gap-4 pl-2 pr-2 sm:pl-3 py-2 rounded-field transition-colors hover:bg-white/5
        ${current ? "bg-primary/8 ring-1 ring-inset ring-primary/25" : ""}`}>
        <CheckToggle checked={checked} onClick={onToggle} label={toggleLabel}/>
        <button className="flex-1 min-w-0 flex items-center gap-3 sm:gap-4 text-left outline-none" onClick={onOpen} aria-label={openLabel}>
            <span className="w-8 shrink-0 font-mono text-sm tabular-nums text-base-content/40">{number}</span>
            <span className="flex-1 min-w-0">
                <span className={`block text-sm font-medium truncate transition-colors group-hover:text-base-content ${checked && !current ? "text-base-content/50" : ""}`}>{title}</span>
                {(progress !== null || progressLabel) && <span className="mt-1.5 flex items-center gap-2.5">
                    {progress !== null && <span className="h-1 w-full max-w-40 rounded-full bg-white/10 overflow-hidden">
                        <span className="block h-full rounded-full bg-primary" style={{ width: `${Math.round(progress * 100)}%` }}></span>
                    </span>}
                    {progressLabel && <span className="shrink-0 text-xs text-base-content/45 tabular-nums">{progressLabel}</span>}
                </span>}
            </span>
            <span className={`grid size-9 shrink-0 place-items-center rounded-full transition-all duration-200
                ${current
                    ? "bg-primary text-primary-content shadow-glow"
                    : "bg-white/8 text-base-content/70 opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 group-hover:bg-primary group-hover:text-primary-content hover-reveal"}`}>
                <OpenIcon size={16} fill={OpenIcon === Play ? "currentColor" : "none"} className={OpenIcon === Play ? "ml-0.5" : ""}/>
            </span>
        </button>
    </li>
}

/**
 * A round watched/read check
 * @param {boolean} checked done
 * @param {function} onClick flips it
 * @param {string} label for screen readers
 */
export function CheckToggle({ checked, onClick, label }) {
    return <button
        type="button"
        aria-pressed={checked}
        aria-label={label}
        onClick={onClick}
        className={`relative grid size-7 shrink-0 place-items-center rounded-full transition-all duration-200
            before:absolute before:-inset-2 before:content-['']
            ${checked
                ? "bg-primary text-primary-content shadow-[0_0_10px_rgb(242_89_18/0.5)]"
                : "ring-1 ring-inset ring-white/20 text-transparent hover:ring-primary/70 hover:text-primary/70"}`}
    >
        <Check size={14} strokeWidth={3}/>
    </button>
}

/**
 * The panel the rows sit in, with an optional header row
 * @param {any} title the list's heading
 * @param {any} actions controls on the right of the heading
 * @param {any} children ItemRow elements
 */
export function ItemList({ title, actions, children }) {
    return <section className="animate-fade-up">
        {(title || actions) && <div className="flex flex-wrap items-center justify-between gap-3 mb-3">
            {typeof title === "string" ? <h2 className="text-xl font-semibold tracking-tight">{title}</h2> : title}
            {actions && <div className="flex items-center gap-2">{actions}</div>}
        </div>}
        <ul className="surface p-1.5 sm:p-2 flex flex-col gap-0.5">
            {children}
        </ul>
    </section>
}
