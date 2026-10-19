import { Clapperboard } from "lucide-react"

/**
 * The MediaHub mark and wordmark
 * @param {boolean} showName shows "MediaHub" beside the mark
 * @param {string} size "sm" for bars, "lg" for the login page
 */
export default function Logo({ showName = true, size = "sm" }) {
    const large = size === "lg"
    return <span className="inline-flex items-center gap-2.5 select-none">
        <span className={`brand-gradient grid place-items-center shrink-0 shadow-glow ${large ? "size-14 rounded-2xl" : "size-9 rounded-xl"}`}>
            <Clapperboard size={large ? 28 : 18} strokeWidth={2.25} className="text-white"/>
        </span>
        {showName && <span className={`font-semibold tracking-tight ${large ? "text-4xl sm:text-5xl" : "text-lg"}`}>
            Media<span className="text-primary">Hub</span>
        </span>}
    </span>
}
