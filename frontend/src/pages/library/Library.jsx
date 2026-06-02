import { useLocation } from "react-router-dom";
import ContentGrid from "../../components/layout/ContentGrid";

export default function Library() {
    const location = useLocation();
    const {items, heading} = location.state ?? {};

    return <ContentGrid items={items} heading={heading}/>
}