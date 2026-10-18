import { Card } from "../cards";
import { CardGridSkeleton } from "../states/Skeletons";
import EmptyState from "../states/EmptyState";

/**
 * A wrapping grid of cards
 * @param {any[]} items the items to show
 * @param {string} heading shown above the grid
 * @param {boolean} loading shows placeholder cards instead of the items
 * @param {string} emptyText what to say when there are no items, nothing is shown without it
 * @param {string} emptyHint what the user can do about it
 */
export default function ContentGrid({items = [], heading, showActions=false, userContentMap={}, onListChange, loading=false, emptyText, emptyHint}) {
    if (loading) return <div className="flex flex-col gap-2">
        {heading && <h2 className="font-bold">{heading}</h2>}
        <CardGridSkeleton/>
    </div>

    if (items.length === 0) {
        return emptyText ? <EmptyState compact title={emptyText} hint={emptyHint}/> : null
    }

    return <div>
        {heading && <h2 className="font-bold mb-2">{heading}</h2>}
        <ul className="flex flex-wrap gap-4">
            {items.map(item => <Card key={item.id ?? item.external_id} item={item} showActions={showActions} userContentMap={userContentMap} onListChange={onListChange}/>)}
        </ul>
    </div>
}
