import { useNavigate } from "react-router-dom"
import { AlertTriangle, ArrowLeft, RotateCw } from "lucide-react"
import { errorMessage } from "../../utils/errors"
import { goBack } from "../../utils/navigation"
import { StateIcon } from "./EmptyState"

/**
 * A failed page or section load, with a way to try again
 * @param {any} error an axios error or a message
 * @param {string} title the headline
 * @param {function} onRetry reloads the data, the button is hidden without it
 * @param {boolean} showBack adds a Back button
 * @param {boolean} compact a smaller version for a section inside a page
 */
export default function ErrorState({ error, title = "Couldn't load this", onRetry, showBack = false, compact = false }) {
    const navigate = useNavigate()

    if (compact) {
        return <div role="alert" className="surface relative flex flex-wrap items-center gap-4 w-full px-5 py-4">
            <span className="absolute left-0 inset-y-3 w-[3px] rounded-r-full bg-error"></span>
            <span className="grid size-10 shrink-0 place-items-center rounded-full bg-error/10 text-error">
                <AlertTriangle size={18}/>
            </span>
            <p className="flex-1 min-w-40 text-sm text-base-content/80">{errorMessage(error)}</p>
            {onRetry && <button className="btn btn-sm btn-ghost" onClick={() => onRetry()}>
                <RotateCw size={14}/> Try again
            </button>}
        </div>
    }

    return <div role="alert" className="flex flex-col items-center text-center gap-3 py-20 px-4 animate-fade-up">
        <StateIcon icon={AlertTriangle} tone="error"/>
        <h2 className="text-xl font-semibold mt-2">{title}</h2>
        <p className="text-base-content/60 max-w-md">{errorMessage(error)}</p>
        <div className="flex gap-2 mt-3">
            {showBack && <button className="btn btn-ghost" onClick={() => goBack(navigate)}>
                <ArrowLeft size={16}/> Back
            </button>}
            {onRetry && <button className="btn btn-primary" onClick={() => onRetry()}>
                <RotateCw size={16}/> Try again
            </button>}
        </div>
    </div>
}
