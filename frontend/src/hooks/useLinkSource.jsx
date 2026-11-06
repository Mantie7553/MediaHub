import { useRef, useState } from "react"
import LinkSourceModal from "../components/modals/LinkSourceModal"
import { searchSuwayomi, suwayomiChapterStats } from "../services/mediaService"

/**
 * Admin: the picker for the Suwayomi source title a manga's missing chapters come from.
 * open() shows it and searches the installed sources for the manga's title
 * @param {object} manga { id, title }
 * @param {object|null} link the current link
 * @param {function} onLinked called with the new link
 * @param {function} onUnlinked called after the link is removed
 * @returns open(), and the modal to render once on the page
 */
export default function useLinkSource({ manga, link, onLinked, onUnlinked }) {
    const dialogRef = useRef(null)
    const [query, setQuery] = useState("")
    const [search, setSearch] = useState(null)
    const [searching, setSearching] = useState(false)
    const [error, setError] = useState(null)
    // chapters looked up after the search, by Suwayomi title id: { loading } | { chapters } | { error }
    const [looked, setLooked] = useState({})
    // a slow answer for an earlier search is ignored
    const latestRef = useRef(0)

    function lookUp(suwayomiId) {
        setLooked(prev => ({ ...prev, [suwayomiId]: { loading: true } }))
        suwayomiChapterStats(manga.id, suwayomiId)
            .then(res => setLooked(prev => ({ ...prev, [suwayomiId]: { chapters: res.data.chapters } })))
            .catch(err => setLooked(prev => ({ ...prev, [suwayomiId]: { error: err } })))
    }

    function run(q) {
        const attempt = latestRef.current + 1
        latestRef.current = attempt
        setSearching(true)
        setError(null)
        searchSuwayomi(manga.id, q)
            .then(res => { if (latestRef.current === attempt) setSearch(res.data) })
            .catch(err => { if (latestRef.current === attempt) { setSearch(null); setError(err) } })
            .finally(() => { if (latestRef.current === attempt) setSearching(false) })
    }

    function open() {
        setQuery(manga.title)
        setSearch(null)
        setLooked({})
        run(manga.title)
        // how far the linked title goes, to see whether it has gone stale
        if (link) lookUp(link.manga_id)
        dialogRef.current?.showModal()
    }

    const modal = <LinkSourceModal
        dialogRef={dialogRef}
        manga={manga}
        link={link}
        query={query}
        onQueryChange={setQuery}
        onSearch={() => run(query)}
        search={search}
        searching={searching}
        error={error}
        looked={looked}
        onLookUp={lookUp}
        onLinked={onLinked}
        onUnlinked={onUnlinked}
    />

    return { open, modal }
}
