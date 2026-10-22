import { useRef, useState } from "react"
import { useNavigate } from "react-router-dom"
import { CollectionCard, NewCollectionCard } from "../cards"
import { CardRowSkeleton } from "../states/Skeletons"
import EmptyState from "../states/EmptyState"
import CollectionNameModal from "../modals/CollectionNameModal"
import { createCollection } from "../../services/collectionsService"
import SectionHeader from "./SectionHeader"
import Shelf from "./Shelf"
import { cardGrid } from "./cardGrid"

// cards a row shows before "See all"
const ROW_LIMIT = 12

/**
 * A row of playlists or collections, ending with a card that makes a new one. A row with more
 * than it shows gets "See all", which opens it up into a grid in place
 * @param {object[]} collections summaries from /me/collections
 * @param {string} kind "playlist" or "collection"
 * @param {string} heading the heading for this row
 * @param {boolean} loading shows placeholder cards instead
 * @param {string} emptyText shown instead of the row when there's nothing and no new card,
 *  e.g. while searching
 */
export default function CollectionList({ collections, kind, heading, loading = false, emptyText }) {
    const navigate = useNavigate()
    const [expanded, setExpanded] = useState(false)
    const newRef = useRef(null)

    if (loading) return <CardRowSkeleton heading={heading} square/>

    function create({ name, description }) {
        return createCollection(kind, name, description || undefined)
            .then(res => navigate(`/${kind === "collection" ? "collections" : "playlists"}/${res.data.id}`))
    }

    const canExpand = collections.length > ROW_LIMIT
    const newCard = !emptyText && <NewCollectionCard kind={kind} fluid={expanded} onClick={() => newRef.current.showModal()}/>

    return <section className="animate-fade-up">
        <SectionHeader title={heading} count={collections.length} expanded={expanded}
            onShowAll={canExpand ? () => setExpanded(!expanded) : undefined}/>
        {collections.length === 0 && emptyText ? (
            <EmptyState compact title={emptyText}/>
        ) : expanded && canExpand ? (
            <ul className={cardGrid}>
                {collections.map(c => <CollectionCard key={c.id} collection={c} fluid/>)}
                {newCard}
            </ul>
        ) : (
            <Shelf>
                {collections.slice(0, ROW_LIMIT).map(c => <CollectionCard key={c.id} collection={c}/>)}
                {newCard}
            </Shelf>
        )}
        <CollectionNameModal dialogRef={newRef} kind={kind} onSave={create}/>
    </section>
}
