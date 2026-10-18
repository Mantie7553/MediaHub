import { Link } from "react-router-dom"
import { Inbox } from "lucide-react"

/**
 * Shown where a list or page has nothing in it yet
 * @param {any} icon a lucide icon component
 * @param {string} title the headline
 * @param {string} hint what the user can do about it
 * @param {object} action optional { label, to } link or { label, onClick } button
 * @param {boolean} compact a smaller dashed box for a row inside a page
 */
export default function EmptyState({ icon: Icon = Inbox, title, hint, action, compact = false }) {
    const button = action && (action.to
        ? <Link to={action.to} className="btn btn-sm btn-primary">{action.label}</Link>
        : <button className="btn btn-sm btn-primary" onClick={action.onClick}>{action.label}</button>)

    if (compact) {
        return <div className="flex flex-col items-center justify-center gap-2 min-h-32 w-full p-4 text-center border border-dashed border-base-300 rounded-lg">
            <p className="text-base-content/60 text-sm">{title}</p>
            {hint && <p className="text-base-content/40 text-xs">{hint}</p>}
            {button}
        </div>
    }

    return <div className="flex flex-col items-center text-center gap-3 py-16 px-4">
        <Icon size={40} className="text-neutral-content"/>
        <h2 className="text-xl font-bold">{title}</h2>
        {hint && <p className="text-neutral-content max-w-md">{hint}</p>}
        {button && <div className="mt-2">{button}</div>}
    </div>
}
