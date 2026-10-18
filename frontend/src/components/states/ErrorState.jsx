import { useNavigate } from "react-router-dom"
import { AlertTriangle, ArrowLeft, RotateCw } from "lucide-react"
import { errorMessage } from "../../utils/errors"
import { goBack } from "../../utils/navigation"

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
        return <div role="alert" className="alert alert-error alert-soft">
            <AlertTriangle size={18}/>
            <span className="text-sm">{errorMessage(error)}</span>
            {onRetry && <button className="btn btn-sm btn-ghost" onClick={() => onRetry()}>
                <RotateCw size={14}/> Try again
            </button>}
        </div>
    }

    return <div role="alert" className="flex flex-col items-center text-center gap-3 py-16 px-4">
        <AlertTriangle size={40} className="text-error"/>
        <h2 className="text-xl font-bold">{title}</h2>
        <p className="text-neutral-content max-w-md">{errorMessage(error)}</p>
        <div className="flex gap-2 mt-2">
            {showBack && <button className="btn btn-ghost" onClick={() => goBack(navigate)}>
                <ArrowLeft size={16}/> Back
            </button>}
            {onRetry && <button className="btn btn-primary" onClick={() => onRetry()}>
                <RotateCw size={16}/> Try again
            </button>}
        </div>
    </div>
}
