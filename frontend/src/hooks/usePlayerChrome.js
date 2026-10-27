import { useCallback, useEffect, useRef, useState } from "react"

// how long the controls stay up without any input while the video plays
const HIDE_AFTER_MS = 3000

/**
 * Whether the player's controls are showing. They show on show() (a tap, the mouse moving, a
 * key), hide on hide() (a tap away from them) and after a few seconds without input
 * @param {boolean} pinned keeps them from hiding by themselves, e.g. while paused or a menu is
 *  open. A tap away still hides them
 * @returns { visible, show, hide }
 */
export default function usePlayerChrome(pinned) {
    const [visible, setVisible] = useState(true)
    const lastInputRef = useRef(0)

    useEffect(() => {
        if (!visible || pinned) return
        // the wait starts over when they're shown or stop being pinned
        lastInputRef.current = Date.now()
        let timer
        function check() {
            const left = HIDE_AFTER_MS - (Date.now() - lastInputRef.current)
            if (left <= 0) setVisible(false)
            else timer = setTimeout(check, left)
        }
        timer = setTimeout(check, HIDE_AFTER_MS)
        return () => clearTimeout(timer)
    }, [visible, pinned])

    const show = useCallback(() => {
        lastInputRef.current = Date.now()
        setVisible(true)
    }, [])
    const hide = useCallback(() => setVisible(false), [])

    return { visible, show, hide }
}
