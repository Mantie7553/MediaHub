import { useCallback, useEffect, useState } from "react"

/**
 * Whether this browser can hand a <video> or <audio> to an AirPlay device (Safari on iPhone,
 * iPad and Mac). Only media the browser plays itself can go to the TV, not media fed through
 * Media Source (hls.js), so the video player plays HLS natively where this is true
 */
export function canAirPlay() {
    return typeof window !== "undefined" && !!window.WebKitPlaybackTargetAvailabilityEvent
}

/**
 * AirPlay for a media element
 * @param {object} mediaRef ref to the <video> or <audio>
 * @param {any} attachKey changes when the element may have been mounted or replaced, so the
 *  listeners move to it
 * @returns { available, active, prompt }: an AirPlay device is on the network, the media is
 *  playing on one now, and a function that opens the device picker
 */
export default function useAirPlay(mediaRef, attachKey) {
    const [available, setAvailable] = useState(false)
    const [active, setActive] = useState(false)

    useEffect(() => {
        const el = mediaRef.current
        if (!el || !canAirPlay()) return
        const onAvailability = e => setAvailable(e.availability === "available")
        const onTarget = () => setActive(!!el.webkitCurrentPlaybackTargetIsWireless)
        el.addEventListener("webkitplaybacktargetavailabilitychanged", onAvailability)
        el.addEventListener("webkitcurrentplaybacktargetiswirelesschanged", onTarget)
        return () => {
            el.removeEventListener("webkitplaybacktargetavailabilitychanged", onAvailability)
            el.removeEventListener("webkitcurrentplaybacktargetiswirelesschanged", onTarget)
        }
    }, [mediaRef, attachKey])

    const prompt = useCallback(() => mediaRef.current?.webkitShowPlaybackTargetPicker?.(), [mediaRef])

    return { available, active, prompt }
}
