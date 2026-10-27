import { useEffect, useState } from "react"
import { cueHtml } from "../../utils/player"

/**
 * Text subtitles drawn by the player rather than the browser, so they can move up out of the
 * way of the controls. The track is left "hidden", which still gives its cues but draws nothing
 * @param {object} videoRef ref to the <video> holding the <track>s
 * @param {string|null} trackUrl the data-url of the <track> to show, null for none
 * @param {boolean} trackReady its <track> is in the video (its file has downloaded)
 * @param {object} picture where the picture is in the player, { left, top, width, height }
 * @param {number} lift pixels to move the subtitles up by
 */
export default function SubtitleOverlay({ videoRef, trackUrl, trackReady, picture, lift }) {
    // the lines showing now, for the track they're from
    const [cues, setCues] = useState({ url: null, html: [] })

    useEffect(() => {
        const element = trackUrl && trackReady
            ? Array.from(videoRef.current?.querySelectorAll("track") ?? []).find(el => el.dataset.url === trackUrl)
            : null
        if (!element) return
        const track = element.track
        function update() {
            setCues({ url: trackUrl, html: Array.from(track.activeCues ?? [], cue => cueHtml(cue.text)) })
        }
        track.addEventListener("cuechange", update)
        // the player turns the track on after this runs, read what's showing once it has
        const frame = requestAnimationFrame(update)
        return () => {
            track.removeEventListener("cuechange", update)
            cancelAnimationFrame(frame)
        }
    }, [videoRef, trackUrl, trackReady])

    const lines = cues.url === trackUrl ? cues.html : []
    if (lines.length === 0) return null

    return <div
        className="absolute z-[5] pointer-events-none flex flex-col items-center justify-end"
        style={{ left: picture.left, top: picture.top, width: picture.width, height: picture.height }}
        aria-live="off"
    >
        <div
            className="flex flex-col items-center gap-1 px-[4%] pb-[5%] text-center font-semibold leading-snug text-white transition-transform duration-200 ease-out [text-shadow:0_0_3px_#000,0_0_3px_#000,0_1px_2px_#000,0_2px_6px_rgb(0_0_0/0.6)]"
            style={{ transform: `translateY(${-lift}px)`, fontSize: `${Math.max(15, Math.min(picture.height * 0.052, 44))}px` }}
        >
            {/* cueHtml leaves only <i>, <b>, <u> and <br> */}
            {lines.map((html, i) => <p key={i} dangerouslySetInnerHTML={{ __html: html }}/>)}
        </div>
    </div>
}
