import { useEffect, useState } from 'react'
import { Navigate } from 'react-router-dom'
import { getAccessToken, refreshSession } from '../../services/session'
import Logo from './Logo'

/**
 * Wrapper to lock users out of a page if they have not logged in. After a reload there is no
 * access token in memory yet, so the session cookie is exchanged for one first
 * @param {any} children
 * @returns
 */
export default function ProtectedRoute({ children }) {
    const [state, setState] = useState(() => getAccessToken() ? "signed-in" : "checking")

    useEffect(() => {
        if (state !== "checking") return
        let cancelled = false
        refreshSession().then(token => {
            if (!cancelled) setState(token ? "signed-in" : "signed-out")
        })
        return () => { cancelled = true }
    }, [state])

    if (state === "signed-out") return <Navigate to="/login" replace />
    if (state === "checking") return <div className="min-h-dvh grid place-items-center" role="status" aria-label="Signing in">
        <div className="animate-pulse"><Logo/></div>
    </div>
    return children
}
