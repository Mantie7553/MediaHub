import { useState } from "react"
import { readStored, writeStored } from "../utils/storage"

/**
 * Whether a row is opened up with "See all". With a key it's remembered in the browser, so the
 * row stays open after a reload or coming back to the page until it's closed again
 * @param {string} [storageKey] names the row, unique on the site. Without one nothing is remembered
 * @returns [expanded, setExpanded]
 */
export default function useExpanded(storageKey) {
    const key = storageKey && `expanded:${storageKey}`
    const [expanded, setExpandedState] = useState(() => key ? readStored(key, "0", ["0", "1"]) === "1" : false)

    function setExpanded(value) {
        setExpandedState(value)
        if (key) writeStored(key, value ? "1" : "0")
    }

    return [expanded, setExpanded]
}
