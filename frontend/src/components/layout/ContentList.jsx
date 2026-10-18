import { useNavigate } from "react-router-dom"
import { Card } from "../cards"
import { CardRowSkeleton } from "../states/Skeletons"
import EmptyState from "../states/EmptyState"

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

    return <div className="my-4">
        <div className="flex justify-between items-center mb-2">
            <h2 className="font-bold">{heading}</h2>
            { items.length > 8 && <button className="link" onClick={handleShowAll}>Show All</button> }
        </div>
        {items.length === 0 ? (
            <EmptyState compact title={emptyText} hint={emptyHint} action={emptyAction}/>
        ) : (
            <>
                <ul className="flex gap-4 overflow-x-auto flex-nowrap">
                    {items.slice(0,9).map(item => {
                        return <Card key={item.id} item={item} userContentMap={userContentMap} onListChange={onListChange}/>
                    })}
                </ul>
                <div className="divider"></div>
            </>
        )}
    </div>
}
