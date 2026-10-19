import { useState } from "react"
import { Eye, EyeOff } from "lucide-react"

/**
 * A labelled text input with an optional icon. Password fields get a show/hide button
 * @param {string} label shown above the input
 * @param {any} icon a lucide icon inside the input
 * @param {string} hint a muted line under the input
 * @param {object} inputProps everything else goes to the <input>
 */
export default function TextField({ label, icon: Icon, hint, type = "text", ...inputProps }) {
    const [shown, setShown] = useState(false)
    const isPassword = type === "password"

    return <label className="flex flex-col gap-1.5">
        {label && <span className="text-xs font-medium uppercase tracking-wider text-base-content/55">{label}</span>}
        <span className="input w-full h-11 bg-base-100/60 border-white/10 focus-within:border-primary/60 focus-within:outline-none focus-within:shadow-[0_0_0_3px_rgb(242_89_18/0.15)] transition-shadow">
            {Icon && <Icon size={17} className="text-base-content/40 shrink-0"/>}
            <input type={isPassword && shown ? "text" : type} className="grow min-w-0 text-base sm:text-sm" {...inputProps}/>
            {isPassword && <button
                type="button"
                className="text-base-content/40 hover:text-base-content transition-colors"
                onClick={() => setShown(!shown)}
                aria-label={shown ? "Hide password" : "Show password"}
            >
                {shown ? <EyeOff size={17}/> : <Eye size={17}/>}
            </button>}
        </span>
        {hint && <span className="text-xs text-base-content/45">{hint}</span>}
    </label>
}

/**
 * A labelled select in the same style as TextField
 * @param {string} label shown above the select
 * @param {any} children the <option> elements
 * @param {object} selectProps everything else goes to the <select>
 */
export function SelectField({ label, children, ...selectProps }) {
    return <label className="flex flex-col gap-1.5">
        <span className="text-xs font-medium uppercase tracking-wider text-base-content/55">{label}</span>
        <select className="select w-full h-11 bg-base-100/60 border-white/10 text-base sm:text-sm" {...selectProps}>{children}</select>
    </label>
}
