import { useEffect, useState } from "react"
import { useSearchParams } from "react-router-dom"

// how long typing pauses before the search goes into the address
const SEARCH_DELAY_MS = 250

/**
 * A search box's text, kept in the address (?q=) once typing pauses, so coming back to the page
 * or refreshing keeps it
 * @returns query (as typed), setQuery, and searchText (trimmed, "" when not searching)
 */
export default function useSearchParamQuery() {
    const [params, setParams] = useSearchParams()
    const [query, setQuery] = useState(params.get("q") ?? "")
    const searchText = query.trim()

    useEffect(() => {
        if (searchText === (params.get("q") ?? "")) return
        const timer = setTimeout(() => {
            // from the address as it is now: React Router hands updaters the last render's params
            const next = new URLSearchParams(window.location.search)
            if (searchText) next.set("q", searchText)
            else next.delete("q")
            setParams(next, { replace: true })
        }, SEARCH_DELAY_MS)
        return () => clearTimeout(timer)
    }, [searchText, params, setParams])

    return { query, setQuery, searchText }
}
