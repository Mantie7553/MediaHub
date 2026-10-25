/**
 * How the chosen subtitle track is shown: "burn" when the server draws it into the video,
 * "ass" when its ASS file is rendered over the video with its styling, otherwise "vtt" through
 * the browser's own captions. ASS falls back to WebVTT until its file has downloaded, when the
 * renderer failed, and while the video element itself is fullscreen, as nothing can be drawn
 * over it then
 * @param {object|null} track the chosen track ({ url, assUrl, burn }), null for off
 * @param {object} options
 * @param {boolean} options.assReady the track's ASS file has downloaded
 * @param {boolean} options.assFailed the ASS renderer couldn't start for it
 * @param {boolean} options.videoFullscreen the video element (not the player) is fullscreen
 * @returns {"off"|"burn"|"ass"|"vtt"}
 */
export function subtitleMode(track, { assReady = false, assFailed = false, videoFullscreen = false } = {}) {
    if (!track) return "off"
    if (track.burn) return "burn"
    if (track.assUrl && assReady && !assFailed && !videoFullscreen && canRenderAss()) return "ass"
    return "vtt"
}

/**
 * Whether this browser has what the ASS renderer needs
 */
export function canRenderAss() {
    return typeof Worker !== "undefined" && typeof WebAssembly !== "undefined" &&
        typeof OffscreenCanvas !== "undefined"
}
