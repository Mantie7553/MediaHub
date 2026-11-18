import { useState } from "react"
import { readStored, writeStored } from "../utils/storage"
import { pickOption } from "../utils/sort"

/**
 * How a list is sorted. With a key it's remembered in the browser, so the list keeps its order
 * after a reload or coming back to the page
 * @param {string} [storageKey] names the list, unique on the site. Without one nothing is remembered
 * @param {object[]} options the list's sort options (see utils/sortOptions); the first is the default
 * @returns [option, setKey]: the picked option, and a setter taking an option's key
 */
export default function useSort(storageKey, options = []) {
    const key = storageKey && `sort:${storageKey}`
    const [picked, setPicked] = useState(() => key ? readStored(key, "") : "")

    function setKey(value) {
        setPicked(value)
        if (key) writeStored(key, value)
    }

    // checked against the options each time, so a key from a list that no longer offers it
    // falls back to the default
    return [pickOption(options, picked), setKey]
}
