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
    // fades the image in once it has loaded instead of painting it in line by line
    const [loadedSrc, setLoadedSrc] = useState(null)

    if (!src || failedSrc === src) {
        return <div className={`${className} bg-linear-to-br from-base-300 to-neutral/60 flex items-center justify-center`} role="img" aria-label={alt || "No cover"}>
            <ImageOff size={28} className="text-neutral-content/60"/>
        </div>
    }

    // MangaDex blocks hotlinking, so its covers are fetched by the backend. A manga from
    // MangaDex can have a cover an admin uploaded, which is served by the backend already
    const url = source === "mangadex" && src.startsWith("https://uploads.mangadex.org/")
        ? `/api/proxy/cover?url=${encodeURIComponent(src)}`
        : src

    return <img
        src={url}
        alt={alt}
        loading="lazy"
        className={`${className} transition-opacity duration-500 ${loadedSrc === src ? "opacity-100" : "opacity-0"}`}
        onLoad={() => setLoadedSrc(src)}
        onError={() => setFailedSrc(src)}
    />
}
