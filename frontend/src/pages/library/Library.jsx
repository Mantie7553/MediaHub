import { useLocation } from "react-router-dom";
import { LibraryBig } from "lucide-react";
import ContentGrid from "../../components/layout/ContentGrid";
import EmptyState from "../../components/states/EmptyState";
import { useUserContent } from "../../hooks";
import PageHeader from "../../components/layout/PageHeader";

/**
 * Every item of one dashboard row ("Show All"). The items come from the dashboard in the
 * navigation state, so a refresh or a shared link has nothing to show
 */
export default function Library() {
    const location = useLocation();
    const {items, heading} = location.state ?? {};
    const { userContentMap, refresh } = useUserContent();

    if (!items) {
        return <EmptyState
            icon={LibraryBig}
            title="Nothing to show here"
            hint="This page lists a row from the dashboard. Open it again with Show All on the dashboard."
            action={{ label: "Go to the dashboard", to: "/" }}
        />
    }

    return <div>
        <PageHeader title={heading} subtitle={`${items.length} ${items.length === 1 ? "title" : "titles"}`}/>
        <ContentGrid items={items} userContentMap={userContentMap} onListChange={refresh} emptyText="Nothing here yet"/>
    </div>
}
