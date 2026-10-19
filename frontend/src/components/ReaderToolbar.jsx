import { ArrowLeft } from "lucide-react"

/**
 * The bar across the top of the manga and light novel readers, with a progress line under it
 * @param {function} onBack leaves the reader
 * @param {string} backLabel where Back goes, like "Chapters"
 * @param {string} title the series
 * @param {string} subtitle the chapter or volume
 * @param {number} progress 0 to 1 through the chapter or volume
 * @param {string} progressLabel like "Page 3 of 20"
 * @param {any} children the reader's own controls, on the right
 */
export default function ReaderToolbar({ onBack, backLabel, title, subtitle, progress = 0, progressLabel, children }) {
    return <header className="sticky top-0 z-20 shrink-0 bg-base-100/80 backdrop-blur-xl border-b border-white/5 pt-[env(safe-area-inset-top)]">
        <div className="flex items-center gap-2 sm:gap-3 h-14 px-2 sm:px-4 max-w-5xl mx-auto">
            <button onClick={onBack} className="btn btn-ghost btn-sm gap-1.5 shrink-0" aria-label={`Back to ${backLabel}`}>
                <ArrowLeft size={18}/>
                <span className="hidden sm:inline">{backLabel}</span>
            </button>
            <div className="flex-1 min-w-0 text-center">
                <p className="text-sm font-medium truncate">{title}</p>
                <p className="text-xs text-base-content/50 truncate tabular-nums">
                    {[subtitle, progressLabel].filter(Boolean).join(" · ")}
                </p>
            </div>
            <div className="flex items-center gap-1 shrink-0">{children}</div>
        </div>
        <div className="h-0.5 bg-white/5">
            <div className="h-full bg-primary shadow-[0_0_8px_rgb(242_89_18/0.8)] transition-[width] duration-300" style={{ width: `${Math.round(Math.min(Math.max(progress, 0), 1) * 100)}%` }}></div>
        </div>
    </header>
}
