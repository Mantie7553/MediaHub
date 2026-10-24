import { useState } from "react"
import { tooltipAlign } from "./chartScale"

/**
 * A line for one series over time, like the library's size per day, over a faint wash down to
 * zero. A crosshair follows the pointer (or the arrow keys) to the nearest day and shows its
 * value; a hidden table carries every value for screen readers
 * @param {string} title what is plotted, shown above the chart
 * @param {object[]} data { key, label, value } in order, at least two
 * @param {any} summary a short note on the right of the title, like the latest value
 * @param {string} color the line's color, any CSS color
 * @param {function} formatValue turns a value into text
 */
export default function LineChart({ title, data, summary, color = "var(--color-primary)", formatValue = v => v.toLocaleString() }) {
    const [active, setActive] = useState(null)
    // a little headroom over the highest point, from a zero baseline so growth isn't exaggerated
    const top = Math.max(1, ...data.map(d => d.value)) * 1.1
    const x = i => data.length > 1 ? (i / (data.length - 1)) * 100 : 50
    const y = v => (1 - v / top) * 100
    const line = data.map((d, i) => `${i === 0 ? "M" : "L"}${x(i)},${y(d.value)}`).join(" ")
    const area = `${line} L${x(data.length - 1)},100 L${x(0)},100 Z`
    const shown = active === null ? null : data[active]

    function pick(e) {
        const rect = e.currentTarget.getBoundingClientRect()
        const at = (e.clientX - rect.left) / rect.width
        setActive(Math.min(data.length - 1, Math.max(0, Math.round(at * (data.length - 1)))))
    }

    return <figure className="min-w-0">
        <figcaption className="flex items-baseline justify-between gap-3 mb-3">
            <span className="text-sm font-medium">{title}</span>
            {summary && <span className="text-xs text-base-content/55 tabular-nums">{summary}</span>}
        </figcaption>

        <div className="grid grid-cols-[auto_1fr] gap-x-2">
            <div className="relative h-36 min-w-10 text-[10px] text-base-content/45 tabular-nums text-right" aria-hidden="true">
                <span className="absolute right-0 -translate-y-1/2 whitespace-nowrap" style={{ top: `${y(top / 1.1)}%` }}>{formatValue(top / 1.1)}</span>
                <span className="absolute right-0 -translate-y-1/2" style={{ top: "100%" }}>{formatValue(0)}</span>
            </div>
            <div className="relative h-36" onPointerLeave={() => setActive(null)}>
                <div className="absolute inset-x-0 h-px bg-white/6" style={{ top: `${y(top / 1.1)}%` }} aria-hidden="true"></div>
                <div className="absolute inset-x-0 bottom-0 h-px bg-white/10" aria-hidden="true"></div>
                <svg className="absolute inset-0 size-full overflow-visible" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
                    <path d={area} fill={color} fillOpacity="0.1"/>
                    <path d={line} fill="none" stroke={color} strokeWidth="2" strokeLinejoin="round" strokeLinecap="round" vectorEffect="non-scaling-stroke"/>
                </svg>
                {/* the latest point carries an end dot, ringed in the surface color */}
                <EndDot left={x(data.length - 1)} top={y(data[data.length - 1].value)} color={color}/>
                {shown && <>
                    <div className="pointer-events-none absolute inset-y-0 w-px bg-base-content/30" style={{ left: `${x(active)}%` }} aria-hidden="true"></div>
                    <EndDot left={x(active)} top={y(shown.value)} color={color}/>
                    <div className={`pointer-events-none absolute z-10 top-0 surface-raised px-2.5 py-1.5 text-xs whitespace-nowrap ${tooltipAlign(active, data.length)}`}
                        style={{ left: `${x(active)}%` }}>
                        <span className="font-semibold tabular-nums">{formatValue(shown.value)}</span>
                        <span className="text-base-content/55"> · {shown.label}</span>
                    </div>
                </>}
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

function EndDot({ left, top, color }) {
    return <span className="pointer-events-none absolute size-2.5 -translate-x-1/2 -translate-y-1/2 rounded-full ring-2 ring-base-200"
        style={{ left: `${left}%`, top: `${top}%`, backgroundColor: color }} aria-hidden="true"></span>
}
