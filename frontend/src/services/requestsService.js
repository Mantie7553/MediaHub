import api from './api'

/**
 * Make an API call to approve a download request
 * @param {any} id the request id
 * @returns
 */
export function approveRequest(id) {
    return api.put(`/requests/${id}/approve`)
}

/**
 * Make an API call to reject a download request
 * @param {any} id the request id
 * @param {any} adminNotes any notes an admin may add
 * @returns
 */
export function rejectRequest(id, adminNotes = "") {
    return api.put(`/requests/${id}/reject`, { admin_notes: adminNotes })
}