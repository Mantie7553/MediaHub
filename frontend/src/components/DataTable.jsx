/**
 * Rows of records: a table when there is room, a stack of cards when there isn't (phones, narrow
 * panels). Room is measured on the table's own container, not the window, so a half width
 * panel on a wide screen gets cards too. Cells can use @2xl: variants to adapt the same way
 * @param {object[]} columns { key, header, render?(value, item), primary?, hideOnMobile?, className? }.
 *  The primary column is the card's title on phones; the "actions" key sits at the card's foot
 * @param {object[]} items the records
 * @param {function} rowKey picks a record's key, its id by default
 */
export default function DataTable({ columns, items, rowKey = item => item.id }) {
    const cell = (col, item) => col.render ? col.render(item[col.key], item) : item[col.key]
    const primary = columns.find(col => col.primary) ?? columns[0]
    const actions = columns.find(col => col.key === "actions")
    const details = columns.filter(col => col !== primary && col !== actions && !col.hideOnMobile)

    return <div className="@container">
        {/* scrolls inside the panel rather than spilling out if a cell is still too wide */}
        <div className="hidden @2xl:block overflow-x-auto">
        <table className="w-full text-sm">
            <thead>
                <tr className="text-left text-xs uppercase tracking-wider text-base-content/45">
                    {columns.map(col => <th key={col.key} className={`font-medium px-3 pb-3 ${col.key === "actions" ? "text-right" : ""}`}>{col.header}</th>)}
                </tr>
            </thead>
            <tbody>
                {items.map(item => (
                    <tr key={rowKey(item)} className="group border-t border-white/5 transition-colors hover:bg-white/[0.03]">
                        {columns.map(col => (
                            <td key={col.key} className={`px-3 py-3 align-middle ${col.key === "actions" ? "text-right" : ""} ${col.className ?? ""}`}>
                                {cell(col, item)}
                            </td>
                        ))}
                    </tr>
                ))}
            </tbody>
        </table>
        </div>

        <ul className="@2xl:hidden flex flex-col gap-2">
            {items.map(item => (
                <li key={rowKey(item)} className="rounded-field bg-white/[0.03] ring-1 ring-white/5 p-3.5 flex flex-col gap-2.5">
                    <div className="font-medium min-w-0">{cell(primary, item)}</div>
                    {details.length > 0 && <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 text-sm">
                        {details.map(col => <div key={col.key} className="contents">
                            <dt className="text-base-content/45">{col.header}</dt>
                            <dd className="min-w-0 text-right">{cell(col, item)}</dd>
                        </div>)}
                    </dl>}
                    {actions && <div className="flex justify-end border-t border-white/5 pt-2.5">{cell(actions, item)}</div>}
                </li>
            ))}
        </ul>
    </div>
}

/**
 * A titled panel holding a DataTable or other content
 * @param {any} title the heading
 * @param {any} icon a lucide icon shown beside the heading
 * @param {any} meta a muted note beside the title, like a count
 * @param {any} description a line under the heading
 * @param {any} actions buttons on the right of the heading
 * @param {any} children the panel body
 */
export function Panel({ title, icon: Icon, meta, description, actions, children }) {
    return <section className="surface p-4 sm:p-6 min-w-0 animate-fade-up">
        <div className="flex flex-wrap items-start justify-between gap-3 mb-4 sm:mb-5">
            <div className="flex items-start gap-3 min-w-0">
                {Icon && <span className="grid size-10 shrink-0 place-items-center rounded-field bg-linear-to-br from-secondary/25 to-primary/20 ring-1 ring-white/10 text-base-content/80">
                    <Icon size={19}/>
                </span>}
                <div className="min-w-0">
                    <h2 className="text-lg font-semibold tracking-tight flex items-baseline gap-2">
                        {title}
                        {meta != null && <span className="text-sm font-normal text-base-content/40 tabular-nums">{meta}</span>}
                    </h2>
                    {description && <p className="text-sm text-base-content/55 mt-0.5">{description}</p>}
                </div>
            </div>
            {actions && <div className="flex items-center gap-2">{actions}</div>}
        </div>
        {children}
    </section>
}
