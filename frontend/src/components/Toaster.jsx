import { AlertCircle, CheckCircle2, X } from "lucide-react"
import useToastStore from "../stores/useToastStore"

/**
 * Shows the toasts from useToastStore in the bottom corner, above the audio player
 */
export default function Toaster() {
    const toasts = useToastStore(state => state.toasts)
    const dismiss = useToastStore(state => state.dismiss)

    if (toasts.length === 0) return null

    return <div className="toast toast-center sm:toast-end toast-bottom z-[60] mb-[calc(9rem+env(safe-area-inset-bottom))] lg:mb-20" role="status" aria-live="polite">
        {toasts.map(t => {
            const error = t.type === "error"
            const Icon = error ? AlertCircle : CheckCircle2
            return <div key={t.id} className="surface-raised animate-toast-in relative flex items-start gap-3 w-80 max-w-[calc(100vw-2rem)] pl-4 pr-2 py-3">
                <span className={`absolute left-0 inset-y-3 w-[3px] rounded-r-full ${error ? "bg-error" : "bg-success"}`}></span>
                <Icon size={18} className={`mt-0.5 shrink-0 ${error ? "text-error" : "text-success"}`}/>
                <span className="flex-1 text-sm whitespace-normal leading-snug">{t.message}</span>
                <button className="btn btn-ghost btn-xs btn-square text-base-content/50" onClick={() => dismiss(t.id)} aria-label="Dismiss">
                    <X size={14}/>
                </button>
            </div>
        })}
    </div>
}
