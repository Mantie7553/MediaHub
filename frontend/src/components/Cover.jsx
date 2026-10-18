import { useState } from "react"
import { ImageOff } from "lucide-react"

/**
 * Cover art with a placeholder when there is no image or it fails to load
 * @param {string} src the image URL
 * @param {string} source the item's external_source, MangaDex covers go through our proxy
 * @param {string} alt describes the image
 * @param {string} className size and fit, used for the placeholder too
 */
export default function Cover({ src, source, alt = "", className = "" }) {
    const [failedSrc, setFailedSrc] = useState(null)

    if (!src || failedSrc === src) {
        return <div className={`${className} bg-base-300 flex items-center justify-center`} role="img" aria-label={alt || "No cover"}>
            <ImageOff size={28} className="text-neutral-content"/>
        </div>
    }

    // MangaDex blocks hotlinking, so its covers are fetched by the backend
    const url = source === "mangadex"
        ? `/api/proxy/cover?url=${encodeURIComponent(src)}`
        : src

    return <img src={url} alt={alt} loading="lazy" className={className} onError={() => setFailedSrc(src)}/>
}
