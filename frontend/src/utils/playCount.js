// listening this long counts as a play, or half the track when it is shorter than a minute
export const PLAY_AFTER_SECS = 30

// a jump forward bigger than this is a seek, not listening
const MAX_STEP_SECS = 2

/**
 * Follow how long a track has actually been listened to, so a play is counted once someone has
 * heard enough of it rather than when they skip past it. Seeking ahead doesn't add listening
 * time; going back to the start after a counted play (repeat, or restarting the track) starts a
 * new play
 * @param {object|null} state what this returned last time, null to start
 * @param {string} trackId the track playing
 * @param {number} time the player's position in seconds
 * @param {number} duration the track's length in seconds, 0 when not known yet
 * @returns { state, counted } where counted is true on the update that makes it a play
 */
export function trackListening(state, trackId, time, duration) {
    if (!state || state.trackId !== trackId) {
        state = { trackId, listened: 0, last: time, counted: false }
    }
    let { listened, last, counted } = state

    if (counted && time < MAX_STEP_SECS && last > time + 5) {
        listened = 0
        counted = false
    }
    const step = time - last
    if (step > 0 && step < MAX_STEP_SECS) listened += step

    const needed = duration > 0 ? Math.min(PLAY_AFTER_SECS, duration / 2) : PLAY_AFTER_SECS
    const nowCounted = !counted && listened >= needed
    return { state: { trackId, listened, last: time, counted: counted || nowCounted }, counted: nowCounted }
}
