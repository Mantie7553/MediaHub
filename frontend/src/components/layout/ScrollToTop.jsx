import { useLayoutEffect } from "react"
import { useLocation, useNavigationType } from "react-router-dom"

/**
 * Starts every newly opened page at the top. The plain router keeps the window's scroll, so a
 * details page opened from far down a list would open part way down. Back and forward are left
 * alone so the browser returns to where the user was. Only the path counts: changing the query
 * on the same page (a season, a tab) doesn't jump
 */
export default function ScrollToTop() {
    const { pathname } = useLocation()
    const navigationType = useNavigationType()

    useLayoutEffect(() => {
        if (navigationType !== "POP") window.scrollTo(0, 0)
    }, [pathname, navigationType])

    return null
}
