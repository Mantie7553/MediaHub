// a jump forward bigger than this is a seek, not watching
const MAX_STEP_SECS = 2

/**
 * How much of the video was watched between two positions the player reported, so time spent
 * watching can be counted rather than how far someone got. Seeking (either way), and the jump
 * when a new stream starts, count as nothing
 * @param {number|null} last the position at the previous update, null when there wasn't one
 * @param {number} time the position now, in seconds
 * @returns {number} seconds watched since the previous update
 */
export function watchedBetween(last, time) {
    if (last === null || !Number.isFinite(last) || !Number.isFinite(time)) return 0
    const step = time - last
    return step > 0 && step <= MAX_STEP_SECS ? step : 0
}
