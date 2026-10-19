import { create } from "zustand"
import { readStored, writeStored } from "../utils/storage"

const STORAGE_KEY = "sidebar_collapsed"

/**
 * Whether the desktop sidebar is shrunk to an icon rail, remembered in this browser
 */
const useSidebarStore = create((set, get) => ({
    collapsed: readStored(STORAGE_KEY, "0") === "1",

    toggle: () => {
        const collapsed = !get().collapsed
        writeStored(STORAGE_KEY, collapsed ? "1" : "0")
        set({ collapsed })
    },
}))

export default useSidebarStore
