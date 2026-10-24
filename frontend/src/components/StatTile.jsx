/**
 * One headline figure, like space left or users active this week
 * @param {any} icon a lucide icon shown beside the label
 * @param {string} label what the figure is
 * @param {any} value the figure
 * @param {any} hint a muted line under it
 * @param {string} tone error or warning colors the figure, for something that needs a look
 */
export default function StatTile({ icon: Icon, label, value, hint, tone }) {
    const color = tone === "error" ? "text-error" : tone === "warning" ? "text-warning" : ""
    return <div className="surface p-4 min-w-0 animate-fade-up">
        <p className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wider text-base-content/50">
            {Icon && <Icon size={14} className="shrink-0"/>}
            <span className="truncate">{label}</span>
        </p>
        <p className={`mt-1.5 text-2xl font-semibold tracking-tight tabular-nums truncate ${color}`}>{value}</p>
        {hint && <p className="mt-0.5 text-xs text-base-content/50 truncate">{hint}</p>}
    </div>
}
