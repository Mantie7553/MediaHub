import { useState } from "react"
import { useSearchParams } from "react-router-dom";
import { BookOpen, Film, Music, Tv } from "lucide-react";
import ErrorState from "../../components/states/ErrorState";
import { CardGridSkeleton } from "../../components/states/Skeletons";
import api from "../../services/api";
import useApi from "../../hooks/useApi";
import ContentList from "../../components/layout/ContentList";
import ContentGrid from "../../components/layout/ContentGrid";
import useUserContent from "../../hooks/useUserContent";
import MusicDiscover from "./MusicDiscover";
import PageHeader from "../../components/layout/PageHeader";
import SegmentedTabs from "../../components/SegmentedTabs";
import SearchField from "../../components/SearchField";
import SectionHeader from "../../components/layout/SectionHeader";
import { readStored, writeStored } from "../../utils/storage";

const emptySections = { trending: [], popular: [], top_rated: [] };

// key is what the URL shows (?type=music), type is what the API calls it
const tabs = [
    { key: "anime", type: "anime", label: "Anime", icon: Tv, noun: "anime" },
    { key: "manga", type: "manga", label: "Manga", icon: BookOpen, noun: "manga" },
    { key: "movie", type: "movie", label: "Movies", icon: Film, noun: "movies" },
    { key: "music", type: "music_track", label: "Music", icon: Music, noun: "music" },
];
const TAB_STORAGE_KEY = "discover_tab";

export default function Discover() {
    const { userContentMap, refresh } = useUserContent();
    // the tab lives in the URL so a reload or Back keeps it, and is remembered for the next
    // visit from the sidebar, which links to plain /discover
    const [searchParams, setSearchParams] = useSearchParams();
    const tabKeys = tabs.map(t => t.key);
    const tabKey = tabKeys.includes(searchParams.get("type"))
        ? searchParams.get("type")
        : readStored(TAB_STORAGE_KEY, "anime", tabKeys);
    const tab = tabs.find(t => t.key === tabKey);
    const activeTab = tab.type;
    const [query, setQuery] = useState("");
    const isMusic = activeTab === "music_track";
    const sections = useApi(isMusic ? null : `/search?type=${activeTab}`, { initial: emptySections });
    const library = useApi(isMusic ? null : `/media?available=true&type=${activeTab}`, { initial: [] });
    // the finished search: its query, results and error. null when not searching
    const [search, setSearch] = useState(null);
    const [searching, setSearching] = useState(false);

    function handleSearch() {
        const q = query.trim();
        if (!q) {
            setSearch(null);
            return;
        }
        setSearching(true);
        api.get(`/search?type=${activeTab}&q=${encodeURIComponent(q)}`)
            .then(resp => setSearch({ query: q, results: resp.data ?? [], error: null }))
            .catch(err => setSearch({ query: q, results: [], error: err }))
            .finally(() => setSearching(false))
    }

    function handleTabChange(key) {
        writeStored(TAB_STORAGE_KEY, key);
        setSearchParams({ type: key }, { replace: true });
        setSearch(null);
        setQuery("");
    }

    const listProps = { showActions: true, userContentMap, onListChange: refresh, loading: sections.loading };
    const topRatedHeading = activeTab === "manga" ? "Latest Updates" : "Top Rated";

    return <div className="flex flex-col gap-10">
        <div>
            <PageHeader title="Discover" subtitle="Find something new to watch, read or listen to."/>
            <SegmentedTabs options={tabs} value={tabKey} onChange={handleTabChange} label="Media type" size="lg"/>
        </div>

        {activeTab === "music_track" ? (
            <MusicDiscover onListChange={refresh}/>
        ) : (
            <>
                <SearchField
                    value={query}
                    onChange={setQuery}
                    onSubmit={handleSearch}
                    onClear={() => { setSearch(null); setQuery(""); }}
                    searching={searching}
                    placeholder={`Search ${tab.noun}…`}
                />

                {/* Available Now */}
                {library.error ? (
                    <section>
                        <SectionHeader title="Available Now"/>
                        <ErrorState compact error={library.error} onRetry={library.reload} />
                    </section>
                ) : (
                    <ContentList items={library.data} heading="Available Now" userContentMap={userContentMap} onListChange={refresh}
                        loading={library.loading} emptyText="Nothing downloaded yet" />
                )}

                {/* Search Results */}
                {searching ? (
                    <section>
                        <SectionHeader title="Search Results"/>
                        <CardGridSkeleton count={6} />
                    </section>
                ) : search ? (
                    search.error
                        ? <ErrorState compact error={search.error} onRetry={handleSearch} />
                        : <ContentGrid items={search.results} heading={`Results for "${search.query}"`} showActions={true} userContentMap={userContentMap} onListChange={refresh}
                            emptyText={`No results for "${search.query}"`} emptyHint="Check the spelling or try the title in another language." />
                ) : sections.error ? (
                    <ErrorState compact error={sections.error} onRetry={sections.reload} />
                ) : (
                    <>
                        <ContentList items={sections.data.trending ?? []} heading="Trending" {...listProps} />
                        <ContentList items={sections.data.popular ?? []} heading="Popular" {...listProps} />
                        <ContentList items={sections.data.top_rated ?? []} heading={topRatedHeading} {...listProps} />
                    </>
                )}
            </>
        )}
    </div>
}