import { useLocation } from "react-router-dom";
import ContentGrid from "../../components/layout/ContentGrid";
import { useUserContent } from "../../hooks";

export default function Library() {
    const location = useLocation();
    const {items, heading} = location.state ?? {};
    const { userContentMap, refresh } = useUserContent();

    return <ContentGrid items={items} heading={heading} userContentMap={userContentMap} onListChange={refresh}/>
}