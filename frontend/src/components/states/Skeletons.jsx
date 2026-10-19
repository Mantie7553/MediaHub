/**
 * Placeholders shaped like the content they stand in for, so pages don't jump when data arrives
 */

/**
 * One card, the same size as components/cards/Card.jsx
 * @param {boolean} square album covers instead of posters
 * @param {boolean} fluid fills its grid cell
 */
export function CardSkeleton({ square = false, fluid = false }) {
    return <li className={`${fluid ? "min-w-0" : "w-[8.5rem] sm:w-40 md:w-44 shrink-0"} flex flex-col gap-2`} aria-hidden="true">
        <div className={`skeleton w-full rounded-box ${square ? "aspect-square" : "aspect-[2/3]"}`}></div>
        <div className="skeleton h-3.5 w-3/4 mt-1"></div>
        <div className="skeleton h-3 w-1/2"></div>
    </li>
}

/**
 * A heading and a row of cards, like components/layout/ContentList.jsx
 * @param {string} heading shown as is, so the page keeps its labels while loading
 * @param {number} count how many cards
 * @param {boolean} square album covers instead of posters
 */
export function CardRowSkeleton({ heading, count = 8, square = false }) {
    return <section aria-busy="true">
        {heading ? <h2 className="text-lg font-semibold tracking-tight mb-3">{heading}</h2> : <div className="skeleton h-5 w-32 mb-3"></div>}
        <ul className="flex gap-3 sm:gap-4 overflow-hidden flex-nowrap pt-1 pb-3">
            {Array.from({ length: count }, (_, i) => <CardSkeleton key={i} square={square}/>)}
        </ul>
    </section>
}

/**
 * A wrapping grid of cards, like components/layout/ContentGrid.jsx
 * @param {number} count how many cards
 */
export function CardGridSkeleton({ count = 12 }) {
    return <ul className="grid grid-cols-[repeat(auto-fill,minmax(9.5rem,1fr))] md:grid-cols-[repeat(auto-fill,minmax(10.5rem,1fr))] gap-x-4 gap-y-6" aria-busy="true">
        {Array.from({ length: count }, (_, i) => <CardSkeleton key={i} fluid/>)}
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

/**
 * A reader toolbar with a progress bar
 */
export function ReaderToolbarSkeleton() {
    return <div className="sticky top-0 z-10 bg-base-200 border-b border-base-300 w-full" aria-hidden="true">
        <div className="max-w-2xl mx-auto px-6 py-3 flex items-center gap-3">
            <div className="skeleton h-8 w-28 shrink-0"></div>
            <div className="flex-1 flex flex-col gap-2 items-center">
                <div className="skeleton h-3 w-12"></div>
                <div className="skeleton h-2 w-full"></div>
            </div>
            <div className="skeleton h-8 w-20 shrink-0"></div>
        </div>
    </div>
}

/**
 * A manga page while its image loads
 */
export function PageSkeleton() {
    return <div className="skeleton mt-2 h-[calc(100vh-5rem)] aspect-[2/3] max-w-full" aria-busy="true" aria-label="Loading page"></div>
}

/**
 * Paragraphs of a light novel while the volume loads
 */
export function TextSkeleton({ paragraphs = 4 }) {
    return <div className="max-w-2xl mx-auto w-full px-6 py-8 flex flex-col gap-6" aria-busy="true" aria-label="Loading text">
        {Array.from({ length: paragraphs }, (_, p) => (
            <div key={p} className="flex flex-col gap-2">
                <div className="skeleton h-4 w-full"></div>
                <div className="skeleton h-4 w-full"></div>
                <div className="skeleton h-4 w-11/12"></div>
                <div className="skeleton h-4 w-3/5"></div>
            </div>
        ))}
    </div>
}

/**
 * A whole reader page: toolbar plus a manga page or light novel text
 * @param {string} variant "page" or "text"
 */
export function ReaderSkeleton({ variant = "page" }) {
    return <div className="flex flex-col">
        <ReaderToolbarSkeleton/>
        {variant === "page"
            ? <div className="flex justify-center"><PageSkeleton/></div>
            : <TextSkeleton/>}
    </div>
}
