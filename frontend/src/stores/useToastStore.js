import { create } from "zustand"
import { errorMessage } from "../utils/errors"

const DISMISS_MS = 5000
let nextId = 1

/**
 * Store for the short messages shown in the corner (see components/Toaster.jsx)
 */
const useToastStore = create((set, get) => ({
    toasts: [],

    // action is an optional { label, onClick } button, such as Undo
    show: (type, message, action = null) => {
        const id = nextId++
        set({ toasts: [...get().toasts, { id, type, message, action }] })
        setTimeout(() => get().dismiss(id), DISMISS_MS)
        return id
    },

    dismiss: (id) => set({ toasts: get().toasts.filter(t => t.id !== id) }),
}))

/**
 * Show a toast from anywhere, including outside components
 */
export const toast = {
    success: (message, action) => useToastStore.getState().show("success", message, action),
    error: (err, fallback) => useToastStore.getState().show("error", errorMessage(err, fallback)),
}

export default useToastStore
