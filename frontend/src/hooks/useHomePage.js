import { useCallback, useEffect } from "react"
import api from "../services/api"
import { setCached } from "../services/apiCache"
import useApi from "./useApi"
import { DEFAULT_HOME, homePage } from "../utils/homePages"

const PATH = "/me/preferences"

// every mounted useHomePage, so picking a new home updates the tabs, the button and Settings
// together
const listeners = new Set()

/**
 * The page the user picked as their home, and a way to change it. It's stored on the server, so
 * it follows them to every device
 * @returns home (a homePages entry), loading (until the setting has loaded the first time),
 *  error, and setHomePage(key), a promise that settles once it's saved
 */
export default function useHomePage() {
    const prefs = useApi(PATH)
    const { reload } = prefs

    useEffect(() => {
        const listener = () => reload(true)
        listeners.add(listener)
        return () => { listeners.delete(listener) }
    }, [reload])

    const setHomePage = useCallback(async key => {
        // the languages are required, and the player may have changed them since this loaded
        const { data: current } = await api.get(PATH)
        const { data: saved } = await api.put(PATH, {
            audio_language: current.audio_language,
            subtitle_language: current.subtitle_language,
            home_page: key,
        })
        setCached(PATH, saved)
        listeners.forEach(listener => listener())
        return saved
    }, [])

    return {
        home: homePage(prefs.data?.home_page ?? DEFAULT_HOME),
        loading: prefs.loading,
        error: prefs.error,
        setHomePage,
    }
}
