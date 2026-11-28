import api from './api'

/**
 * Mark one of the user's notifications read, or unread again
 * @param {string} id the notification id
 * @param {boolean} read true for read, false for unread
 * @returns the response, with its new read_at
 */
export function markNotificationRead(id, read = true) {
    return api.put(`/me/notifications/${id}/read`, { read })
}

/**
 * Mark every unread notification of the user's read
 * @returns the response, with how many changed
 */
export function markAllNotificationsRead() {
    return api.post('/me/notifications/read-all')
}

/**
 * Which kinds of notification the user gets
 * @returns the response, kind → on
 */
export function getNotificationPrefs() {
    return api.get('/me/notification-prefs')
}

/**
 * Turn kinds of notification on or off
 * @param {Record<string, boolean>} prefs kind → on, the ones to change
 * @returns the response, every kind → on
 */
export function setNotificationPrefs(prefs) {
    return api.put('/me/notification-prefs', prefs)
}
