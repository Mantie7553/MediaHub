import { BadgeCheck } from "lucide-react"

/**
 * The "E" streaming apps put beside explicit tracks
 */
export function ExplicitBadge() {
    return <span className="inline-grid place-items-center size-4 shrink-0 rounded-[3px] bg-base-content/20 text-[10px] font-bold leading-none text-base-content/80"
        title="Explicit" aria-label="Explicit">E</span>
}

/**
 * Who a music search result is by. Songs show "Artist · Album"; videos show the channel that
 * posted them, with a check when it's the official upload or a verified channel
 * @param {object} track a music search result
 * @param {string} className classes for the line
 */
export default function TrackByline({ track, className = "" }) {
    const isSong = track.kind === "song"
    const by = (isSong ? track.artist : track.uploader) || track.artist || track.uploader || "Unknown"
    const mark = !isSong && (track.official ? "Official upload" : track.verified ? "Verified channel" : null)
    return <p className={`flex items-center gap-1 min-w-0 ${className}`}>
        <span className="truncate">
            {by}
            {isSong && track.album && <span className="text-base-content/40"> · {track.album}</span>}
        </span>
        {mark && <BadgeCheck size={13} className="shrink-0 text-primary" aria-label={mark}><title>{mark}</title></BadgeCheck>}
    </p>
}
