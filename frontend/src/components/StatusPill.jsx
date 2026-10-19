const dots = {
    info: "bg-info",
    success: "bg-success",
    warning: "bg-warning",
    error: "bg-error",
    primary: "bg-primary",
    secondary: "bg-secondary",
}

/**
 * A small rounded label with a colored dot, for statuses like airing, pending or downloading
 * @param {string} label the text
 * @param {string} tone info, success, warning, error, primary, secondary, or anything else for grey
 * @param {boolean} pulse the dot pulses, for something in progress
 */
export default function StatusPill({ label, tone, pulse = false }) {
    const dot = dots[tone] ?? "bg-neutral-content"
    return <span className="inline-flex items-center gap-1.5 whitespace-nowrap rounded-full bg-white/8 ring-1 ring-white/10 px-2.5 py-0.5 text-xs font-medium capitalize text-base-content/90">
        <span className="relative flex size-1.5">
            {pulse && <span className={`absolute inset-0 rounded-full ${dot} animate-ping opacity-75`}></span>}
            <span className={`relative size-1.5 rounded-full ${dot}`}></span>
        </span>
        {label}
    </span>
}
