import { X } from "lucide-react"
import useToastStore from "../stores/useToastStore"

/**
 * Shows the toasts from useToastStore in the bottom corner, above the audio player
 */
export default function Toaster() {
    const toasts = useToastStore(state => state.toasts)
    const dismiss = useToastStore(state => state.dismiss)

    if (toasts.length === 0) return null

    return <div className="toast toast-end toast-bottom z-50 mb-20" role="status" aria-live="polite">
        {toasts.map(t => (
            <div key={t.id} className={`alert ${t.type === "error" ? "alert-error" : "alert-success"} shadow-lg max-w-sm`}>
                <span className="text-sm whitespace-normal">{t.message}</span>
                <button className="btn btn-ghost btn-xs btn-circle" onClick={() => dismiss(t.id)} aria-label="Dismiss">
                    <X size={14}/>
                </button>
            </div>
        ))}
    </div>
}
