import { ChevronRight } from "lucide-react"

/**
 * The heading over a row or grid of content
 * @param {string} title the heading
 * @param {number} count shown muted beside the heading
 * @param {function} onShowAll adds a "See all" link
 * @param {any} children extra controls on the right
 */
export default function SectionHeader({ title, count, onShowAll, children }) {
    return <div className="flex items-end justify-between gap-4 mb-3">
        <h2 className="text-lg font-semibold tracking-tight flex items-baseline gap-2">
            {title}
            {count > 0 && <span className="text-sm font-normal text-base-content/40 tabular-nums">{count}</span>}
        </h2>
        <div className="flex items-center gap-2">
            {children}
            {onShowAll && <button
                className="group/all flex items-center gap-0.5 text-sm text-base-content/60 hover:text-primary transition-colors"
                onClick={onShowAll}
            >
                See all <ChevronRight size={16} className="transition-transform group-hover/all:translate-x-0.5"/>
            </button>}
        </div>
    </div>
}
