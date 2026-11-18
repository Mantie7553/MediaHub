import useExpanded from "../../hooks/useExpanded"
import useSort from "../../hooks/useSort"
import { sortItems } from "../../utils/sort"
import { SortMenu } from "../details/Menus"
import { AlbumCard } from "../cards"
import EmptyState from "../states/EmptyState"
import { CardRowSkeleton } from "../states/Skeletons"
import SectionHeader from "./SectionHeader"
import Shelf from "./Shelf"
import { cardGrid } from "./cardGrid"

// cards a row shows before "See all"
const ROW_LIMIT = 12

/**
 * A row of album cards with a play button on each. A row with more than it shows gets
 * "See all", which opens it up into a grid of every album in place
 * @param {any[]} albums the albums to show
 * @param {string} heading the heading for this list
 * @param {boolean} loading shows placeholder cards instead of the albums
 * @param {string} emptyText what to say when there are no albums
 * @param {string} emptyHint what the user can do about it
 * @param {object} emptyAction optional { label, to } shown when there are no albums
 * @param {string} expandKey remembers "See all" and the sort across reloads under this name
 * @param {object[]} sortOptions the ways the row can be sorted (see utils/sortOptions), none
 *  leaves it in the order given
 * @param {object} userContentMap the user's list entries, for sorting by their rating
 */
export default function AlbumList({ albums: given, heading, loading = false, emptyText = "Nothing here yet", emptyHint, emptyAction, expandKey, sortOptions, userContentMap }) {
    const [expanded, setExpanded] = useExpanded(expandKey)
    const [sort, setSort] = useSort(expandKey, sortOptions)

    if (loading) return <CardRowSkeleton heading={heading} square/>

    const albums = sortItems(given, sort, { userContentMap })
    if (!albums || albums.length === 0) {
        return <section>
            <SectionHeader title={heading}/>
            <EmptyState compact title={emptyText} hint={emptyHint} action={emptyAction}/>
        </section>
    }

    const canExpand = albums.length > ROW_LIMIT

    return <section className="animate-fade-up">
        <SectionHeader title={heading} count={albums.length} expanded={expanded}
            onShowAll={canExpand ? () => setExpanded(!expanded) : undefined}>
            {sortOptions && albums.length > 1 && <SortMenu options={sortOptions} value={sort} onChange={setSort} label={`Sort ${heading}`}/>}
        </SectionHeader>
        {expanded && canExpand ? (
            <ul className={cardGrid}>{albums.map(album => <AlbumCard key={album.id} album={album} fluid/>)}</ul>
        ) : (
            <Shelf>{albums.slice(0, ROW_LIMIT).map(album => <AlbumCard key={album.id} album={album}/>)}</Shelf>
        )}
    </section>
}
