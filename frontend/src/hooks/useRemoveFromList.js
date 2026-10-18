import { removeFromList } from '../services/mediaService'
import { toast } from '../stores/useToastStore'

/**
 * Take a title off the user's list from a details page, with a toast either way
 * @param {object} userEntry the user's list entry, nothing happens without one
 * @param {function} onRemoved reloads the user's lists afterwards
 * @returns a function that removes the entry
 */
export default function useRemoveFromList(userEntry, onRemoved) {
    return function remove() {
        if (!userEntry) return
        removeFromList(userEntry.id)
            .then(() => { toast.success("Removed from your list"); onRemoved?.() })
            .catch(err => toast.error(err, "Could not remove it from your list"))
    }
}
