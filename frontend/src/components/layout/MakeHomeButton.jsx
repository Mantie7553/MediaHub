import { useState } from "react"
import { House } from "lucide-react"
import useHomePage from "../../hooks/useHomePage"
import { homePage } from "../../utils/homePages"
import { toast } from "../../stores/useToastStore"

/**
 * Makes this page the one "/" opens, or says it already is
 * @param {string} pageKey the home page's key
 */
export default function MakeHomeButton({ pageKey }) {
    const { home, loading, setHomePage } = useHomePage()
    const [saving, setSaving] = useState(false)

    if (loading) return <span className="skeleton h-8 w-36 rounded-full"></span>

    if (home.key === pageKey) {
        return <span className="inline-flex items-center gap-1.5 h-8 px-3 rounded-full text-xs font-medium text-primary bg-primary/12">
            <House size={14}/> Your home
        </span>
    }

    function makeHome() {
        setSaving(true)
        setHomePage(pageKey)
            .then(() => toast.success(`${homePage(pageKey).label} is now your home page`))
            .catch(err => toast.error(err, "Couldn't change your home page"))
            .finally(() => setSaving(false))
    }

    return <button className="btn btn-sm btn-ghost rounded-full bg-white/5 gap-1.5" onClick={makeHome} disabled={saving}>
        <House size={14}/> Make this my home
    </button>
}
