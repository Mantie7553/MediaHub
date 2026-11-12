import { useState } from "react";
import ErrorState from "../../components/states/ErrorState";
import SegmentedTabs from "../../components/SegmentedTabs";

/**
 * A titled group of rows. With more than one row it gets filter chips; under "All" empty rows are
 * skipped, and an empty section shows one message instead of one per row
 * @param {object[]} rows { key, label, count, render, error?, reload? }, error and reload for rows
 *  that load separately
 * @param {any} error the whole section failed to load
 * @param {function} onRetry tries the whole section again
 * @param {any} empty shown under "All" when every row is empty
 * @param {any} actions links or buttons beside the title
 */
export default function HomeSection({ title, rows, loading, error, onRetry, empty, actions }) {
    const [filter, setFilter] = useState("all");
    const filtered = rows.length > 1 && filter !== "all";

    const visible = filtered
        ? rows.filter(row => row.key === filter)
        : rows.filter(row => loading || row.error || row.count > 0);

    // rows sharing one failed request show its error once
    const shownErrors = new Set();

    return <section className="flex flex-col gap-6">
        <div className="flex flex-col sm:flex-row sm:flex-wrap sm:items-center justify-between gap-3 sm:gap-4 border-b border-white/5 pb-4">
            <div className="flex items-center justify-between gap-3 min-w-0">
                <h2 className="text-xl sm:text-2xl font-semibold tracking-tight flex items-center gap-3">
                    <span className="w-1 h-6 rounded-full bg-primary shadow-[0_0_10px_rgb(242_89_18/0.7)]"></span>
                    {title}
                </h2>
                {actions && <div className="flex items-center gap-2 sm:hidden">{actions}</div>}
            </div>
            <div className="flex items-center gap-3 min-w-0">
                {rows.length > 1 && <SegmentedTabs
                    options={[{ key: "all", label: "All" }, ...rows.map(row => ({ key: row.key, label: row.label, count: loading ? 0 : row.count }))]}
                    value={filter}
                    onChange={setFilter}
                    label={`Filter ${title}`}
                />}
                {actions && <div className="hidden sm:flex items-center gap-2 shrink-0">{actions}</div>}
            </div>
        </div>

        {error ? (
            <ErrorState compact error={error} onRetry={onRetry}/>
        ) : visible.length === 0 ? (
            empty
        ) : visible.map(row => {
            if (row.error) {
                if (shownErrors.has(row.error)) return null;
                shownErrors.add(row.error);
                return <ErrorState key={row.key} compact error={row.error} onRetry={row.reload}/>;
            }
            return <div key={row.key}>{row.render()}</div>;
        })}
    </section>
}
