import axios from 'axios'
import { clearCached } from './apiCache'

const BASE_URL = import.meta.env.VITE_API_URL

// the access token lives only in memory, so a script injected into the page can't read it out
// of storage. The session itself is an HttpOnly cookie the browser sends to /api/auth, which
// scripts can't read at all; a reload gets a new access token from it
let accessToken = null
let refreshing = null

// tells other tabs when this one signs out, so they don't keep working until their token expires
const channel = typeof BroadcastChannel !== 'undefined' ? new BroadcastChannel('mediahub-session') : null
channel?.addEventListener('message', e => {
    if (e.data === 'signed-out') {
        forgetSession()
        window.location.href = '/login'
    }
})

/**
 * The access token for API requests, null before signing in or after a reload until the
 * session has been refreshed
 */
export function getAccessToken() {
    return accessToken
}

/**
 * Keep the access token a sign in, sign up or refresh answered with
 * @param {string} token the JWT
 */
export function setAccessToken(token) {
    accessToken = token
}

/**
 * Role of the signed in user, read from the access token
 * @returns "admin", "user", or null when not signed in
 */
export function getRole() {
    if (!accessToken) return null
    try {
        return JSON.parse(atob(accessToken.split('.')[1])).role
    } catch {
        return null
    }
}

/**
 * Get a new access token from the session cookie. Calls made while one is already running
 * share it, so a burst of expired requests refreshes once
 * @returns the new token, or null when there is no valid session
 */
export function refreshSession() {
    if (!refreshing) {
        // browsers signed in before the cookie existed send their stored token once, then the
        // server has moved it into the cookie and it is deleted from storage
        const legacy = localStorage.getItem('refresh_token')
        refreshing = axios.post(`${BASE_URL}/auth/refresh`, legacy ? { refresh_token: legacy } : {})
            .then(res => {
                accessToken = res.data.token
                return accessToken
            })
            .catch(() => {
                accessToken = null
                return null
            })
            .finally(() => {
                localStorage.removeItem('token')
                localStorage.removeItem('refresh_token')
                refreshing = null
            })
    }
    return refreshing
}

/**
 * Forget the access token and everything loaded with it, in this tab
 */
export function forgetSession() {
    accessToken = null
    clearCached()
}

/**
 * Sign out: end the session on the server (best effort), forget it here and in other tabs
 * @param {boolean} everywhere end every session of this user, on all devices
 */
export function endSession(everywhere = false) {
    const request = everywhere
        ? axios.post(`${BASE_URL}/auth/logout-all`, null, { headers: { Authorization: `Bearer ${accessToken}` } })
        : axios.delete(`${BASE_URL}/auth/logout`)
    forgetSession()
    channel?.postMessage('signed-out')
    return request.catch(() => {})
}
