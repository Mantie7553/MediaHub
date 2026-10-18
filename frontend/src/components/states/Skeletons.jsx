/**
 * Placeholders shaped like the content they stand in for, so pages don't jump when data arrives
 */

/**
 * One card, the same size as components/cards/Card.jsx
 */
export function CardSkeleton() {
    return <li className="w-44 shrink-0 flex flex-col gap-2" aria-hidden="true">
        <div className="skeleton h-56 w-full"></div>
        <div className="skeleton h-4 w-3/4"></div>
        <div className="skeleton h-3 w-1/2"></div>
    </li>
}

/**
 * A heading and a row of cards, like components/layout/ContentList.jsx
 * @param {string} heading shown as is, so the page keeps its labels while loading
 * @param {number} count how many cards
 */
export function CardRowSkeleton({ heading, count = 6 }) {
    return <div className="my-4" aria-busy="true">
        {heading ? <h2 className="font-bold mb-2">{heading}</h2> : <div className="skeleton h-5 w-32 mb-2"></div>}
        <ul className="flex gap-4 overflow-hidden flex-nowrap">
            {Array.from({ length: count }, (_, i) => <CardSkeleton key={i}/>)}
        </ul>
    </div>
}

/**
 * A wrapping grid of cards, like components/layout/ContentGrid.jsx
 * @param {number} count how many cards
 */
export function CardGridSkeleton({ count = 12 }) {
    return <ul className="flex flex-wrap gap-4" aria-busy="true">
        {Array.from({ length: count }, (_, i) => <CardSkeleton key={i}/>)}
    </ul>
}

/**
 * A details page: cover on the left, title and text on the right, a list below
 */
export function DetailsSkeleton() {
    return <div className="flex flex-col gap-8" aria-busy="true">
        <div className="skeleton h-8 w-20"></div>
        <div className="flex gap-6">
            <div className="skeleton w-48 h-64 shrink-0"></div>
            <div className="flex flex-col gap-3 w-full max-w-xl">
                <div className="skeleton h-8 w-2/3"></div>
                <div className="skeleton h-4 w-24"></div>
                <div className="flex gap-1">
                    <div className="skeleton h-5 w-16"></div>
                    <div className="skeleton h-5 w-16"></div>
                </div>
                <div className="skeleton h-4 w-full"></div>
                <div className="skeleton h-4 w-full"></div>
                <div className="skeleton h-4 w-4/5"></div>
                <div className="skeleton h-10 w-32 mt-2"></div>
            </div>
        </div>
        <ListSkeleton/>
    </div>
}

/**
 * Rows of a list, like episodes or chapters
 * @param {number} rows how many rows
 */
export function ListSkeleton({ rows = 5 }) {
    return <div className="flex flex-col gap-2" aria-busy="true">
        {Array.from({ length: rows }, (_, i) => <div key={i} className="skeleton h-12 w-full"></div>)}
    </div>
}

/**
 * A table with a header and some rows
 * @param {number} columns how many columns
 * @param {number} rows how many rows
 */
export function TableSkeleton({ columns = 4, rows = 4 }) {
    return <div className="flex flex-col gap-3 min-w-80" aria-busy="true">
        <div className="flex gap-4">
            {Array.from({ length: columns }, (_, i) => <div key={i} className="skeleton h-4 flex-1"></div>)}
        </div>
        {Array.from({ length: rows }, (_, i) => <div key={i} className="skeleton h-8 w-full"></div>)}
    </div>
}
