import { useState } from "react"
import { Card } from "../cards"
import { CardRowSkeleton } from "../states/Skeletons"
import EmptyState from "../states/EmptyState"
import SectionHeader from "./SectionHeader"
import Shelf from "./Shelf"
import { cardGrid } from "./cardGrid"

// cards a row shows before "See all"
const ROW_LIMIT = 12

/**
 * A list of some content type. A row with more than it shows gets "See all", which opens it
 * up into a grid of every item in place
 * @param {any} items the items this list will contain
 * @param {any} heading the heading for this list
 * @param {boolean} loading shows placeholder cards instead of the items
 * @param {string} emptyText what to say when there are no items
 * @param {string} emptyHint what the user can do about it
 * @param {object} emptyAction optional { label, to } shown when there are no items
 * @returns
 */
export default function ContentList({items, heading, userContentMap={}, onListChange, loading=false, emptyText="Nothing here yet", emptyHint, emptyAction}) {
    const [expanded, setExpanded] = useState(false);

    if (loading) return <CardRowSkeleton heading={heading}/>

    const canExpand = items.length > ROW_LIMIT;
    const card = (item, fluid) => <Card key={item.id} item={item} userContentMap={userContentMap} onListChange={onListChange} fluid={fluid}/>

    return <section className="animate-fade-up">
        <SectionHeader title={heading} count={items.length} expanded={expanded}
            onShowAll={canExpand ? () => setExpanded(!expanded) : undefined}/>
        {items.length === 0 ? (
            <EmptyState compact title={emptyText} hint={emptyHint} action={emptyAction}/>
        ) : expanded && canExpand ? (
            <ul className={cardGrid}>{items.map(item => card(item, true))}</ul>
        ) : (
            <Shelf>{items.slice(0, ROW_LIMIT).map(item => card(item, false))}</Shelf>
        )}
    </section>
}
