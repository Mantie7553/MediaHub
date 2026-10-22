import { useCallback, useEffect, useRef, useState } from "react"
import { hideOnScroll } from "../utils/search"

/**
 * Hide something while the page scrolls down and bring it back when it scrolls up or is near
 * the top
 * @param {boolean} pinned keeps it showing and ignores scrolling meanwhile, e.g. while typing,
 *  when a phone's keyboard opening scrolls the page
 * @returns hidden, and show() to bring it back by hand
 */
export default function useHideOnScroll(pinned = false) {
    const [hidden, setHidden] = useState(false)
    // shared with the scroll listener, so show() and the listener agree on what's showing
    const hiddenRef = useRef(false)
    const pinnedRef = useRef(pinned)

    useEffect(() => {
        pinnedRef.current = pinned
    }, [pinned])

    useEffect(() => {
        let fromY = window.scrollY
        function onScroll() {
            const y = window.scrollY
            if (pinnedRef.current) {
                fromY = y
                return
            }
            const next = hideOnScroll(hiddenRef.current, fromY, y)
            if (next.moved) fromY = y
            if (next.hidden !== hiddenRef.current) {
                hiddenRef.current = next.hidden
                setHidden(next.hidden)
            }
        }
        window.addEventListener("scroll", onScroll, { passive: true })
        return () => window.removeEventListener("scroll", onScroll)
    }, [])

    const show = useCallback(() => {
        hiddenRef.current = false
        setHidden(false)
    }, [])

    return { hidden: hidden && !pinned, show }
}
