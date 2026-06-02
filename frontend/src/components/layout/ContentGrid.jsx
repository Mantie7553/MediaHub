import { Card } from "../cards";

export default function ContentGrid({items, heading, showActions=false}) {
    return items.length > 0 ? 
    <div>
        <h2 className="font-bold">{heading}</h2>
        <ul className="flex flex-wrap gap-4">
            {items.map(item => <Card key={item.id ?? item.external_id} item={item} showActions={showActions}/>)}
        </ul>
    </div> :
    <></>
}