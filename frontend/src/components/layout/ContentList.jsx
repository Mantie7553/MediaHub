import { useNavigate } from "react-router-dom"
import { Card } from "../cards"

/**
 * A list of some content type
 * @param {any} items the items this list will contain
 * @param {any} heading the heading for this list
 * @returns
 */
export default function ContentList({items, heading, userContentMap={}, onListChange}) {
    const navigate = useNavigate();

    function handleShowAll() {
        navigate("/library", { state: {items, heading}});
    }

    return <div className="my-4">
        <div className="flex justify-between items-center mb-2">
            <h2 className="font-bold">{heading}</h2>
            { items.length > 8 && <button className="link" onClick={handleShowAll}>Show All</button> }
        </div>
        {items.length === 0 ? (
            <div className="flex items-center justify-center h-32 w-full border border-dashed border-base-300 rounded-lg">
                <p className="text-base-content/50 text-sm pl-2">Nothing here yet</p>
            </div>
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