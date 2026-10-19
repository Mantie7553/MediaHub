import { useState } from "react"
import { createPortal } from "react-dom"

/**
 * A label to the right of an icon in the collapsed sidebar. It is drawn on <body> at a fixed
 * position, because the sidebar scrolls and would clip a tooltip drawn inside it
 * @param {string} label the text to show
 * @param {boolean} enabled only the desktop icon rail needs it
 * @param {any} children the icon link or button
 */
export default function RailTooltip({ label, enabled, children }) {
    const [anchor, setAnchor] = useState(null)

    function show(e) {
        // the mobile drawer always shows labels, even when the desktop rail is collapsed
        if (!enabled || !window.matchMedia("(min-width: 1024px)").matches) return
        const rect = e.currentTarget.getBoundingClientRect()
        setAnchor({ left: rect.right + 12, top: rect.top + rect.height / 2 })
    }

    const hide = () => setAnchor(null)

    return <div onMouseEnter={show} onMouseLeave={hide} onFocus={show} onBlur={hide}>
        {children}
        {enabled && anchor && createPortal(
            <div
                role="tooltip"
                className="fixed z-[100] -translate-y-1/2 pointer-events-none animate-fade-in whitespace-nowrap rounded-field surface-raised px-2.5 py-1.5 text-sm font-medium"
                style={{ left: anchor.left, top: anchor.top }}
            >
                {label}
            </div>,
            document.body
        )}
    </div>
}
