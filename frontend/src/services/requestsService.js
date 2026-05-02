import api from './api'

export function approveRequest(id) {
    return api.put(`/requests/${id}/approve`)
}

export function rejectRequest(id, adminNotes = "") {
    return api.put(`/requests/${id}/reject`, { admin_notes: adminNotes })
}