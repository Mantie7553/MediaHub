import { useState } from "react"
import { Bookmark, BookmarkCheck } from "lucide-react"
import useWishlistStore, { addToWishlist, removeFromWishlist } from "../stores/useWishlistStore"
import { wishlistedId } from "../utils/wishlist"
import { toast } from "../stores/useToastStore"

/**
 * Puts a title on the user's wishlist, which asks for it to be downloaded, or takes it off. It
 * sits beside the list status and never changes it
 * @param {object} item the title: a library title, list entry or search result
 * @param {function} onChange called after it was added or removed
 * @param {string} className the button's look, the details pages' action style by default
 */
export default function WishlistButton({ item, onChange, className = "btn border-white/10 bg-white/10 hover:bg-white/15 backdrop-blur gap-2 font-medium" }) {
    const keys = useWishlistStore(state => state.keys)
    const wishlisted = wishlistedId(keys, item)
    const [busy, setBusy] = useState(false)

    function toggle(e) {
        e.stopPropagation()
        e.preventDefault()
        setBusy(true)
        const request = wishlisted
            ? removeFromWishlist(wishlisted).then(() => toast.success("Removed from your wishlist"))
            : addToWishlist(item).then(() => toast.success("Added to your wishlist, download requested"))
        request
            .then(() => onChange?.())
            .catch(err => toast.error(err, "Could not update your wishlist"))
            .finally(() => setBusy(false))
    }

    return <button type="button" className={`${className} ${wishlisted ? "text-primary" : ""}`} onClick={toggle} disabled={busy}
        aria-pressed={!!wishlisted}>
        {busy
            ? <span className="loading loading-spinner loading-xs"/>
            : wishlisted ? <BookmarkCheck size={16}/> : <Bookmark size={16}/>}
        {wishlisted ? "Wishlisted" : "Wishlist"}
    </button>
}
