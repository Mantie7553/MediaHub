import { create } from "zustand"

const STORAGE_KEY = "sidebar_collapsed"

function readCollapsed() {
    try {
        return localStorage.getItem(STORAGE_KEY) === "1"
    } catch {
        return false
    }
}

/**
 * Whether the desktop sidebar is shrunk to an icon rail, remembered in this browser
 */
const useSidebarStore = create((set, get) => ({
    collapsed: readCollapsed(),

    toggle: () => {
        const collapsed = !get().collapsed
        try {
            localStorage.setItem(STORAGE_KEY, collapsed ? "1" : "0")
        } catch {
            // private windows can refuse storage, the toggle still works for this visit
        }
        set({ collapsed })
    },
}))

export default useSidebarStore
