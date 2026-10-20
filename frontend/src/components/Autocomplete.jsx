import { useId, useState } from "react"
import TextField from "./TextField"
import { suggestions } from "../utils/suggest"

/**
 * A TextField that suggests matching options while typing. Arrow keys move through them and
 * Enter picks the highlighted one; with nothing highlighted Enter submits the form as usual.
 * Escape closes the list without closing a dialog around it. The list sits in the page flow
 * under the field, so it is never cut off inside a dialog
 * @param {string} label shown above the input
 * @param {string} value the text in the box
 * @param {function} onChange called with the new text, typed or picked
 * @param {string[]} options everything that can be suggested
 * @param {string} hint a muted line under the input
 * @param {object} inputProps everything else goes to the <input>
 */
export default function Autocomplete({ label, value, onChange, options, hint, ...inputProps }) {
    const listId = useId()
    const [open, setOpen] = useState(false)
    const [highlight, setHighlight] = useState(-1)
    const matches = open ? suggestions(options, value) : []
    const shown = matches.length > 0

    function pick(option) {
        onChange(option)
        setOpen(false)
        setHighlight(-1)
    }

    function handleKeyDown(e) {
        if (e.key === "ArrowDown" || e.key === "ArrowUp") {
            e.preventDefault()
            if (!open) { setOpen(true); return }
            const step = e.key === "ArrowDown" ? 1 : -1
            setHighlight(h => matches.length ? (h + step + matches.length) % matches.length : -1)
        } else if (e.key === "Enter" && shown && highlight >= 0) {
            e.preventDefault()
            pick(matches[highlight])
        } else if (e.key === "Escape" && shown) {
            e.preventDefault()
            e.stopPropagation()
            setOpen(false)
        }
    }

    return <div className="flex flex-col">
        <TextField
            label={label}
            hint={hint}
            value={value}
            onChange={e => { onChange(e.target.value); setOpen(true); setHighlight(-1) }}
            onFocus={() => setOpen(true)}
            onBlur={() => { setOpen(false); setHighlight(-1) }}
            onKeyDown={handleKeyDown}
            role="combobox"
            autoComplete="off"
            aria-autocomplete="list"
            aria-expanded={shown}
            aria-controls={listId}
            aria-activedescendant={shown && highlight >= 0 ? `${listId}-${highlight}` : undefined}
            {...inputProps}
        />
        {shown && <ul id={listId} role="listbox" aria-label={label ? `${label} suggestions` : "Suggestions"}
            className="mt-1.5 surface-raised p-1 flex flex-col gap-0.5 max-h-52 overflow-y-auto animate-fade-up">
            {matches.map((option, i) => (
                <li
                    key={option}
                    id={`${listId}-${i}`}
                    role="option"
                    aria-selected={i === highlight}
                    // keeps focus in the input, so the list doesn't close before the click lands
                    onMouseDown={e => e.preventDefault()}
                    onClick={() => pick(option)}
                    onMouseEnter={() => setHighlight(i)}
                    className={`px-3 py-2 rounded-field text-sm cursor-pointer truncate transition-colors
                        ${i === highlight ? "bg-primary/15 text-base-content" : "text-base-content/75 hover:bg-white/5"}`}
                >
                    {option}
                </li>
            ))}
        </ul>}
    </div>
}
