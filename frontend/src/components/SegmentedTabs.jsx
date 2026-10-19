/**
 * A row of pill tabs where one is selected. Scrolls sideways on narrow screens instead of wrapping
 * @param {object[]} options { key, label, icon?, count? }, icon is a lucide component
 * @param {string} value the selected key
 * @param {function} onChange called with the picked key
 * @param {string} label names the tab list for screen readers
 * @param {string} size "md", or "lg" for a page's main tabs
 */
export default function SegmentedTabs({ options, value, onChange, label, size = "md" }) {
    const large = size === "lg"

    return <div className="shelf -mx-4 px-4 sm:mx-0 sm:px-0 overflow-x-auto scroll-px-4 sm:scroll-px-0">
        <div role="tablist" aria-label={label}
            className={`inline-flex gap-1 rounded-full bg-base-200 ring-1 ring-white/5 shadow-card ${large ? "p-1.5" : "p-1"}`}>
            {options.map(({ key, label, icon: Icon, count }) => {
                const active = value === key
                return <button
                    key={key}
                    type="button"
                    role="tab"
                    aria-selected={active}
                    onClick={() => onChange(key)}
                    className={`shrink-0 flex items-center gap-2 rounded-full whitespace-nowrap transition-all duration-200
                        ${large ? "h-10 px-5 text-sm" : "h-8 px-3.5 text-sm"}
                        ${active
                            ? "bg-primary text-primary-content font-medium shadow-glow"
                            : "text-base-content/65 hover:text-base-content hover:bg-white/5"}`}
                >
                    {Icon && <Icon size={large ? 17 : 15} strokeWidth={active ? 2.25 : 1.75}/>}
                    {label}
                    {count > 0 && <span className={`text-xs tabular-nums ${active ? "text-primary-content/75" : "text-base-content/40"}`}>{count}</span>}
                </button>
            })}
        </div>
    </div>
}
