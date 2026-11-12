import { create } from "zustand"

/**
 * Store for the one rating dialog (components/RatingPrompt.jsx). It lives with the app rather
 * than with the card or page that asks, so a card leaving its row once the title is completed
 * doesn't take the dialog with it
 */
const useRatingStore = create(set => ({
    // { entryId, title, rating, prompt, onSaved, onDismiss }, a new object for each ask
    request: null,
    ask: request => set({ request: { ...request } }),
}))

/**
 * Open the rating dialog from anywhere
 * @param {object} request entryId (the list entry), title, rating (now, null for none), prompt
 *  (asked because it was just completed), onSaved (after a save), onDismiss (closed unsaved)
 */
export function rateTitle(request) {
    useRatingStore.getState().ask(request)
}

export default useRatingStore
