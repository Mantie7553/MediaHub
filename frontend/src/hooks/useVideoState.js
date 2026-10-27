import { useEffect, useState } from "react"
import { bufferedEnd } from "../utils/player"

const INITIAL = { currentTime: 0, duration: 0, buffered: 0, paused: true, muted: false, volume: 1, rate: 1, videoWidth: 0, videoHeight: 0 }
const EVENTS = ["timeupdate", "durationchange", "loadedmetadata", "progress", "play", "pause", "volumechange", "ratechange", "seeking", "emptied", "resize"]

/**
 * What the video is doing, for the player's own controls
 * @param {object} videoRef ref to the <video>, which stays mounted
 * @returns { currentTime, duration, buffered, paused, muted, volume, rate, videoWidth, videoHeight }
 */
export default function useVideoState(videoRef) {
    const [state, setState] = useState(INITIAL)

    useEffect(() => {
        const video = videoRef.current
        if (!video) return
        function update() {
            setState({
                currentTime: video.currentTime,
                duration: Number.isFinite(video.duration) ? video.duration : 0,
                buffered: bufferedEnd(video.buffered, video.currentTime),
                paused: video.paused,
                muted: video.muted,
                volume: video.volume,
                rate: video.playbackRate,
                videoWidth: video.videoWidth,
                videoHeight: video.videoHeight,
            })
        }
        for (const event of EVENTS) video.addEventListener(event, update)
        return () => {
            for (const event of EVENTS) video.removeEventListener(event, update)
        }
    }, [videoRef])

    return state
}
