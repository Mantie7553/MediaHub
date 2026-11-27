import { create } from "zustand"
import api from "../services/api"

/**
 * How many of the user's notifications are unread, for the bell. refresh() asks the server
 * again; a failed refresh keeps the last count
 */
const useNotificationStore = create(set => ({
    unread: 0,

    refresh: () => api.get("/me/notifications/unread-count")
        .then(res => set({ unread: res.data?.count ?? 0 }))
        .catch(() => {}),
}))

export default useNotificationStore
