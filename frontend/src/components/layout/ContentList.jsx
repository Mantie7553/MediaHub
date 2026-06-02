import { useNavigate } from "react-router-dom"
import { Card } from "../cards"

/**
 * A list of some content type
 * @param {any} items the items this list will contain
 * @param {any} heading the heading for this list
 * @returns
 */
export default function ContentList({items, heading}) {
    const navigate = useNavigate();

    function handleShowAll() {
        navigate("/library", { state: {items, heading}});
    }

    return <div className="my-4 max-w-fit">
        <div className="flex justify-between items-center mb-2">
            {items.length > 0 && <h2 className="font-bold">{heading}</h2> }
            { items.length > 10 && <button className="link" onClick={handleShowAll}>Show All</button> }
        </div>
        <ul className="flex gap-4 overflow-x-auto flex-nowrap">
            {items.slice(0,8).map(item => {
                return <Card key={item.id} item={item} />
            })}
        </ul>
    </div>
}