import { useNavigate } from "react-router-dom"
import { ArrowLeft, Home, SearchX } from "lucide-react"
import { goBack } from "../../utils/navigation"
import { StateIcon } from "./EmptyState"
import useDocumentTitle from "../../hooks/useDocumentTitle"

/**
 * A page or item that doesn't exist
 * @param {string} title the headline
 * @param {string} hint more detail
 */
export default function NotFound({ title = "Page not found", hint = "This link doesn't lead anywhere. It may have been mistyped or removed." }) {
    const navigate = useNavigate()
    useDocumentTitle(title)

    return <div className="flex flex-col items-center text-center gap-3 py-20 px-4 animate-fade-up">
        <StateIcon icon={SearchX}/>
        <h2 className="text-xl font-semibold mt-2">{title}</h2>
        <p className="text-base-content/60 max-w-md">{hint}</p>
        <div className="flex gap-2 mt-3">
            <button className="btn btn-ghost" onClick={() => goBack(navigate)}>
                <ArrowLeft size={16}/> Back
            </button>
            <button className="btn btn-primary" onClick={() => navigate("/")}>
                <Home size={16}/> Home
            </button>
        </div>
    </div>
}
