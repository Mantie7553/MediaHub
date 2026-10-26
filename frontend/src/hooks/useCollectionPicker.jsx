import { useEffect, useRef, useState } from "react"
import AddToCollectionModal from "../components/modals/AddToCollectionModal"
import { listCollections } from "../services/collectionsService"
import { addBody } from "../utils/collections"

/**
 * The "Add to playlist / collection" picker, for any page or row: open(item) shows it for that
 * item, loading the user's playlists and collections fresh each time so the ticks are right
 * @param {function} onAdded called after something was added, to reload what's on screen
 * @returns open(item), and the modal to render once on the page
 */
export default function useCollectionPicker(onAdded) {
    const dialogRef = useRef(null)
    const [item, setItem] = useState(null)
    const [options, setOptions] = useState(null)
    const [loadError, setLoadError] = useState(null)
    // bumped on every open, so the dialog opens after the item it's for has rendered, and a slow
    // answer for an earlier open is ignored
    const [opened, setOpened] = useState(0)
    const latestRef = useRef(0)

    useEffect(() => {
        if (opened > 0) dialogRef.current?.showModal()
    }, [opened])

    function open(next) {
        const attempt = latestRef.current + 1
        latestRef.current = attempt
        setItem(next)
        setOptions(null)
        setLoadError(null)
        setOpened(attempt)
        const { season_number: season, ...ids } = addBody(next)
        listCollections(undefined, Object.values(ids)[0], season)
            .then(res => { if (latestRef.current === attempt) setOptions(res.data ?? []) })
            .catch(err => { if (latestRef.current === attempt) setLoadError(err) })
    }

    const modal = <AddToCollectionModal dialogRef={dialogRef} item={item} options={options} loadError={loadError} onAdded={onAdded}/>

    return { open, modal }
}
