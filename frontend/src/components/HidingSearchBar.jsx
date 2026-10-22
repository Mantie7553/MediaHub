import { useRef, useState } from "react"
import { Search } from "lucide-react"
import SearchField from "./SearchField"
import useHideOnScroll from "../hooks/useHideOnScroll"

/**
 * A search box that sticks under the top of the screen, slides away while the page scrolls down
 * and comes back when it scrolls up. While it's away a small search button stays in the corner;
 * tapping it brings the box back, ready to type in
 * @param {string} value the text in the box
 * @param {function} onChange called with the new text
 * @param {string} placeholder hint text
 * @param {string} className extra classes for the bar, e.g. margins. It sticks only within its
 *  parent, so it belongs directly in the page
 */
export default function HidingSearchBar({ value, onChange, placeholder, className = "" }) {
    const inputRef = useRef(null)
    const [focused, setFocused] = useState(false)
    const { hidden, show } = useHideOnScroll(focused)

    function reveal() {
        show()
        // focus without scrolling, the box is sliding back into view on its own
        inputRef.current?.focus({ preventScroll: true })
    }

    return <>
        {/* under the phone header (3.5rem), at the very top on desktop */}
        <div className={`sticky z-20 top-[calc(3.5rem+env(safe-area-inset-top))] lg:top-0 -mx-4 sm:-mx-6 lg:-mx-10 px-4 sm:px-6 lg:px-10 py-3
            bg-base-100/75 backdrop-blur-xl transition-[translate,opacity] duration-300 ease-out ${className}
            ${hidden ? "-translate-y-[calc(100%+1rem)] opacity-0 pointer-events-none" : ""}`}
            aria-hidden={hidden}
        >
            <SearchField
                value={value}
                onChange={onChange}
                onClear={() => onChange("")}
                placeholder={placeholder}
                inputRef={inputRef}
                onFocus={() => setFocused(true)}
                onBlur={() => setFocused(false)}
            />
        </div>
        <button
            type="button"
            onClick={reveal}
            className={`fixed z-20 right-4 top-[calc(4.25rem+env(safe-area-inset-top))] lg:top-4 lg:right-6 btn btn-circle btn-primary shadow-glow
                transition-[opacity,scale] duration-300 ${hidden ? "opacity-100 scale-100" : "opacity-0 scale-75 pointer-events-none"}`}
            aria-label="Show search"
            tabIndex={hidden ? 0 : -1}
        >
            <Search size={18}/>
        </button>
    </>
}
