import { Search, X } from "lucide-react"

/**
 * A large search box with a submit button inside it
 * @param {string} value the text in the box
 * @param {function} onChange called with the new text
 * @param {function} onSubmit runs the search (Enter or the button). Without it the box filters
 *  as it's typed in, there's no button, and Enter just closes a phone's keyboard
 * @param {function} onClear shows a clear button while there is text or a search
 * @param {boolean} searching shows a spinner and disables the button
 * @param {string} placeholder hint text
 * @param {object} inputRef given the input, e.g. to focus it
 * @param {function} onFocus called when the input gains focus
 * @param {function} onBlur called when the input loses focus
 * @param {any} children extra controls between the input and the button
 */
export default function SearchField({ value, onChange, onSubmit, onClear, searching = false, placeholder, inputRef, onFocus, onBlur, children }) {
    function handleSubmit(e) {
        e.preventDefault()
        if (onSubmit) onSubmit()
        else e.currentTarget.querySelector("input")?.blur()
    }

    return <form
        role="search"
        onSubmit={handleSubmit}
        className={`group/search flex items-center gap-2 h-14 w-full max-w-3xl pl-4 rounded-2xl surface ${onSubmit ? "pr-2" : "pr-3"}
            transition-shadow focus-within:shadow-[inset_0_0_0_1px_rgb(242_89_18/0.6),0_0_0_4px_rgb(242_89_18/0.12)]`}
    >
        <Search size={20} className="shrink-0 text-base-content/40 transition-colors group-focus-within/search:text-primary"/>
        <input
            ref={inputRef}
            type="search"
            enterKeyHint="search"
            className="flex-1 min-w-0 h-full bg-transparent outline-none text-base placeholder:text-base-content/35 [&::-webkit-search-cancel-button]:hidden"
            placeholder={placeholder}
            value={value}
            onChange={e => onChange(e.target.value)}
            onFocus={onFocus}
            onBlur={onBlur}
            aria-label={placeholder}
        />
        {onClear && value && <button type="button" className="btn btn-ghost btn-sm btn-circle text-base-content/50" onClick={onClear} aria-label="Clear search">
            <X size={16}/>
        </button>}
        {children}
        {onSubmit && <button type="submit" className="btn btn-primary rounded-xl px-3 sm:px-5" disabled={searching} aria-label="Search">
            {searching ? <span className="loading loading-spinner loading-xs"/> : <Search size={16} className="sm:hidden"/>}
            <span className="hidden sm:inline">Search</span>
        </button>}
    </form>
}
