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
