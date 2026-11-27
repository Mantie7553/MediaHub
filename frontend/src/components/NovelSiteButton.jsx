import { ExternalLink } from "lucide-react"
import { openNovelSite } from "../utils/novelSource"
import { toast } from "../stores/useToastStore"

/**
 * Opens the site light novels are downloaded from, with the title copied to find it there
 * @param {string} title the light novel's title
 * @param {string} className the button's look, the details pages' action style by default
 */
export default function NovelSiteButton({ title, className = "btn border-white/10 bg-white/10 hover:bg-white/15 backdrop-blur gap-2 font-medium" }) {
    function open(e) {
        e.stopPropagation()
        e.preventDefault()
        openNovelSite(title).then(copied => {
            if (copied) toast.success("Title copied: press Ctrl+F on elscione to find it")
        })
    }

    return <button type="button" className={className} onClick={open}>
        <ExternalLink size={16}/>
        Find on elscione
    </button>
}
