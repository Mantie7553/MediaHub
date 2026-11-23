import { useEffect, useRef } from "react"
import { useLocation } from "react-router-dom"
import useAudioStore from "../stores/useAudioStore"
import { CURRENT_BUILD, fetchBuildId, isNewBuild } from "../utils/appUpdate"

/**
 * Reloads a tab that's still running an older build than the server has. Phones keep tabs alive
 * for days, so without this a deploy only shows up after a manual refresh. It checks when the
 * tab is opened or comes back to the front, and reloads straight away unless music is playing;
 * then it waits for the next page change while paused, so it never cuts a song off. The queue and
 * position are saved on the way out, so nothing is lost
 */
export default function useAppUpdate() {
    const { pathname } = useLocation()
    const outdatedRef = useRef(false)

    function reloadIfIdle() {
        if (outdatedRef.current && !useAudioStore.getState().isPlaying) window.location.reload()
    }

    useEffect(() => {
        if (!CURRENT_BUILD) return
        let cancelled = false
        async function check() {
            if (document.visibilityState !== "visible") return
            const latest = await fetchBuildId()
            if (cancelled || !isNewBuild(CURRENT_BUILD, latest)) return
            outdatedRef.current = true
            reloadIfIdle()
        }
        check()
        document.addEventListener("visibilitychange", check)
        return () => {
            cancelled = true
            document.removeEventListener("visibilitychange", check)
        }
    }, [])

    // a page change is a natural moment for a reload that had to wait
    useEffect(reloadIfIdle, [pathname])
}
