import { Card } from "../cards";
import { CardGridSkeleton } from "../states/Skeletons";
import EmptyState from "../states/EmptyState";
import SectionHeader from "./SectionHeader";

/**
 * A wrapping grid of cards
 * @param {any[]} items the items to show
 * @param {string} heading shown above the grid
 * @param {boolean} loading shows placeholder cards instead of the items
 * @param {string} emptyText what to say when there are no items, nothing is shown without it
 * @param {string} emptyHint what the user can do about it
 */
export default function ContentGrid({items = [], heading, showActions=false, userContentMap={}, onListChange, loading=false, emptyText, emptyHint}) {
    if (loading) return <section>
        {heading && <SectionHeader title={heading}/>}
        <CardGridSkeleton/>
    </section>

    if (items.length === 0) {
        return emptyText ? <EmptyState compact title={emptyText} hint={emptyHint}/> : null
    }

    return <section className="animate-fade-up">
        {heading && <SectionHeader title={heading} count={items.length}/>}
        <ul className="grid grid-cols-[repeat(auto-fill,minmax(9.5rem,1fr))] md:grid-cols-[repeat(auto-fill,minmax(10.5rem,1fr))] gap-x-4 gap-y-6">
            {items.map(item => <Card key={item.id ?? item.external_id} item={item} showActions={showActions} userContentMap={userContentMap} onListChange={onListChange} fluid/>)}
        </ul>
    </section>
}
