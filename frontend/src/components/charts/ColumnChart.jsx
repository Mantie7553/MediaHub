import { useState } from "react"
import { niceMax, tooltipAlign } from "./chartScale"

/**
 * Columns for one series over time, like people active per day. Each column shows its value on
 * hover, tap or keyboard focus; a hidden table carries every value for screen readers
 * @param {string} title what is plotted, shown above the chart
 * @param {object[]} data { key, label, value } in order
 * @param {any} summary a short note on the right of the title, like the latest value
 * @param {string} color the columns' background class
 * @param {function} formatValue turns a value into text
 */
export default function ColumnChart({ title, data, summary, color = "bg-primary", formatValue = v => v.toLocaleString() }) {
    const [active, setActive] = useState(null)
    const top = niceMax(Math.max(0, ...data.map(d => d.value)))
    const ticks = top % 2 === 0 ? [top, top / 2, 0] : [top, 0]
    const shown = active === null ? null : data[active]

    function pick(e) {
        const rect = e.currentTarget.getBoundingClientRect()
        setActive(Math.min(data.length - 1, Math.max(0, Math.floor(((e.clientX - rect.left) / rect.width) * data.length))))
    }

    return <figure className="min-w-0">
        <figcaption className="flex items-baseline justify-between gap-3 mb-3">
            <span className="text-sm font-medium">{title}</span>
            {summary && <span className="text-xs text-base-content/55 tabular-nums">{summary}</span>}
        </figcaption>

        <div className="grid grid-cols-[auto_1fr] gap-x-2">
            {/* y axis: clean numbers against recessive gridlines */}
            <div className="relative h-36 w-6 text-[10px] text-base-content/45 tabular-nums" aria-hidden="true">
                {ticks.map(t => <span key={t} className="absolute right-0 -translate-y-1/2" style={{ top: `${(1 - t / top) * 100}%` }}>{formatValue(t)}</span>)}
            </div>
            <div className="relative h-36" onPointerLeave={() => setActive(null)}>
                {ticks.map(t => <div key={t} className="absolute inset-x-0 h-px bg-white/6" style={{ top: `${(1 - t / top) * 100}%` }} aria-hidden="true"></div>)}
                <div className="absolute inset-0 flex items-end gap-0.5" aria-hidden="true">
                    {data.map((d, i) => (
                        <div key={d.key} className="relative flex-1 h-full flex items-end justify-center">
                            {d.value > 0 && <div className={`w-full max-w-6 rounded-t-[4px] ${color} transition-[filter] ${active === i ? "brightness-125" : ""}`}
                                style={{ height: `${(d.value / top) * 100}%` }}></div>}
                        </div>
                    ))}
                </div>
                {shown && <div className={`pointer-events-none absolute z-10 surface-raised px-2.5 py-1.5 text-xs whitespace-nowrap ${tooltipAlign(active, data.length)}`}
                    style={{ left: `${((active + 0.5) / data.length) * 100}%`, bottom: `calc(${(shown.value / top) * 100}% + 6px)` }}>
                    <span className="font-semibold tabular-nums">{formatValue(shown.value)}</span>
                    <span className="text-base-content/55"> · {shown.label}</span>
                </div>}
                {/* one layer over the columns finds the nearest one for the pointer, and steps
                    through them with the arrow keys */}
                <div className="absolute inset-0 rounded-sm focus-visible:outline-2 focus-visible:outline-primary/60" tabIndex={0}
                    role="img" aria-label={`${title}, ${data.length} days`}
                    onFocus={() => setActive(a => a ?? data.length - 1)} onBlur={() => setActive(null)}
                    onKeyDown={e => {
                        if (e.key === "ArrowLeft") setActive(a => Math.max(0, (a ?? data.length) - 1))
                        else if (e.key === "ArrowRight") setActive(a => Math.min(data.length - 1, (a ?? -1) + 1))
                    }}
                    onPointerMove={pick} onPointerDown={pick}></div>
            </div>
            <div></div>
            <div className="flex justify-between mt-1.5 text-[10px] text-base-content/45" aria-hidden="true">
                <span>{data[0]?.label}</span>
                <span>{data[data.length - 1]?.label}</span>
            </div>
        </div>

        <table className="sr-only">
            <caption>{title}</caption>
            <tbody>{data.map(d => <tr key={d.key}><th scope="row">{d.label}</th><td>{formatValue(d.value)}</td></tr>)}</tbody>
        </table>
    </figure>
}
