import { AlertCircle, AlertTriangle, CheckCircle2, X } from "lucide-react"
import useToastStore from "../stores/useToastStore"

const TOAST_LOOKS = {
    success: { icon: CheckCircle2, bar: "bg-success", text: "text-success" },
    warning: { icon: AlertTriangle, bar: "bg-warning", text: "text-warning" },
    error: { icon: AlertCircle, bar: "bg-error", text: "text-error" },
}

/**
 * Shows the toasts from useToastStore in the bottom corner, above the audio player
 */
export default function Toaster() {
    const toasts = useToastStore(state => state.toasts)
    const dismiss = useToastStore(state => state.dismiss)

    if (toasts.length === 0) return null

    return <div className="toast toast-center sm:toast-end toast-bottom z-[60] mb-[calc(5rem+var(--player-h,0px)+env(safe-area-inset-bottom))] lg:mb-[calc(1rem+var(--player-h,0px))]" role="status" aria-live="polite">
        {toasts.map(t => {
            const look = TOAST_LOOKS[t.type] ?? TOAST_LOOKS.success
            const Icon = look.icon
            return <div key={t.id} className="surface-raised animate-toast-in relative flex items-start gap-3 w-80 max-w-[calc(100vw-2rem)] pl-4 pr-2 py-3">
                <span className={`absolute left-0 inset-y-3 w-[3px] rounded-r-full ${look.bar}`}></span>
                <Icon size={18} className={`mt-0.5 shrink-0 ${look.text}`}/>
                <span className="flex-1 text-sm whitespace-normal leading-snug">{t.message}</span>
                {t.action && <button className="btn btn-ghost btn-xs text-primary -my-0.5" onClick={() => { dismiss(t.id); t.action.onClick() }}>
                    {t.action.label}
                </button>}
                <button className="btn btn-ghost btn-xs btn-square text-base-content/50" onClick={() => dismiss(t.id)} aria-label="Dismiss">
                    <X size={14}/>
                </button>
            </div>
        })}
    </div>
}
