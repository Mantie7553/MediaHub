import { useNavigate } from "react-router-dom"
import { Card } from "../cards"
import { CardRowSkeleton } from "../states/Skeletons"
import EmptyState from "../states/EmptyState"
import SectionHeader from "./SectionHeader"
import Shelf from "./Shelf"

/**
 * A list of some content type
 * @param {any} items the items this list will contain
 * @param {any} heading the heading for this list
 * @param {boolean} loading shows placeholder cards instead of the items
 * @param {string} emptyText what to say when there are no items
 * @param {string} emptyHint what the user can do about it
 * @param {object} emptyAction optional { label, to } shown when there are no items
 * @returns
 */
export default function ContentList({items, heading, userContentMap={}, onListChange, loading=false, emptyText="Nothing here yet", emptyHint, emptyAction}) {
    const navigate = useNavigate();

    if (loading) return <CardRowSkeleton heading={heading}/>

    function handleShowAll() {
        navigate("/library", { state: {items, heading}});
    }

    return <section className="animate-fade-up">
        <SectionHeader title={heading} count={items.length} onShowAll={items.length > 12 ? handleShowAll : undefined}/>
        {items.length === 0 ? (
            <EmptyState compact title={emptyText} hint={emptyHint} action={emptyAction}/>
        ) : (
            <Shelf>
                {items.slice(0,12).map(item => {
                    return <Card key={item.id} item={item} userContentMap={userContentMap} onListChange={onListChange}/>
                })}
            </Shelf>
        )}
    </section>
}
