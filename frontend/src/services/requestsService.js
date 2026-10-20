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

/**
 * Archive download requests, hiding them from the Downloads list without deleting them, or
 * bring archived ones back
 * @param {string[]} ids the request ids
 * @param {boolean} archived true to archive, false to restore
 * @returns the response, with how many changed
 */
export function setRequestsArchived(ids, archived) {
    return api.post('/requests/archive', { ids, archived })
}

/**
 * Archive download jobs, or bring archived ones back
 * @param {string[]} ids the job ids
 * @param {boolean} archived true to archive, false to restore
 * @returns the response, with how many changed
 */
export function setJobsArchived(ids, archived) {
    return api.post('/admin/jobs/archive', { ids, archived })
}
