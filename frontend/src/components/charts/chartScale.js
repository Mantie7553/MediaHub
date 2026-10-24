/**
 * The top of a chart's scale: the smallest 1, 2 or 5 times a power of ten at or above max, so
 * the axis shows clean numbers
 * @param {number} max the largest value plotted
 */
export function niceMax(max) {
    if (!(max > 0)) return 1
    const power = 10 ** Math.floor(Math.log10(max))
    for (const step of [1, 2, 5, 10]) {
        if (step * power >= max) return step * power
    }
    return 10 * power
}

/**
 * How to place a tooltip over point i of n so it stays inside the chart: hanging right of
 * points near the left edge, left of ones near the right edge, centered otherwise
 */
export function tooltipAlign(i, n) {
    const at = (i + 0.5) / n
    if (at < 0.15) return "-translate-x-3"
    if (at > 0.85) return "-translate-x-[calc(100%-0.75rem)]"
    return "-translate-x-1/2"
}
