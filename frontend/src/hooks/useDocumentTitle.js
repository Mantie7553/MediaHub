import { useEffect } from "react"

/**
 * Name the browser tab (and its history entry) after the page, "Frieren · MediaHub"
 * @param {string} title the page's title; anything that isn't text leaves just "MediaHub"
 */
export default function useDocumentTitle(title) {
    useEffect(() => {
        const previous = document.title
        document.title = typeof title === "string" && title.trim() ? `${title.trim()} · MediaHub` : "MediaHub"
        return () => { document.title = previous }
    }, [title])
}
