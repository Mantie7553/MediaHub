import { Link } from "react-router-dom"
import { Inbox } from "lucide-react"

/**
 * Shown where a list or page has nothing in it yet
 * @param {any} icon a lucide icon component
 * @param {string} title the headline
 * @param {string} hint what the user can do about it
 * @param {object} action optional { label, to } link or { label, onClick } button
 * @param {boolean} compact a smaller panel for a row inside a page
 */
export default function EmptyState({ icon: Icon = Inbox, title, hint, action, compact = false }) {
    const button = action && (action.to
        ? <Link to={action.to} className="btn btn-sm btn-primary">{action.label}</Link>
        : <button className="btn btn-sm btn-primary" onClick={action.onClick}>{action.label}</button>)

    if (compact) {
        return <div className="surface flex flex-wrap items-center gap-4 w-full px-5 py-4">
            <span className="grid size-10 shrink-0 place-items-center rounded-full bg-white/5 text-base-content/40">
                <Icon size={18}/>
            </span>
            <div className="flex-1 min-w-40">
                <p className="text-sm font-medium text-base-content/80">{title}</p>
                {hint && <p className="text-xs text-base-content/50 mt-0.5">{hint}</p>}
            </div>
            {button}
        </div>
    }

    return <div className="flex flex-col items-center text-center gap-3 py-20 px-4 animate-fade-up">
        <StateIcon icon={Icon}/>
        <h2 className="text-xl font-semibold mt-2">{title}</h2>
        {hint && <p className="text-base-content/60 max-w-md">{hint}</p>}
        {button && <div className="mt-3">{button}</div>}
    </div>
}

/**
 * The large icon on a full page state, in a soft glowing circle
 * @param {any} icon a lucide icon component
 * @param {string} tone "neutral" or "error"
 */
export function StateIcon({ icon: Icon, tone = "neutral" }) {
    const error = tone === "error"
    return <span className={`relative grid size-20 place-items-center rounded-full ring-1
        ${error ? "bg-error/10 ring-error/20 text-error" : "bg-linear-to-br from-secondary/20 to-primary/15 ring-white/10 text-base-content/70"}`}>
        <span className={`absolute inset-0 rounded-full blur-2xl ${error ? "bg-error/20" : "bg-secondary/20"}`}></span>
        <Icon size={34} strokeWidth={1.5} className="relative"/>
    </span>
}
