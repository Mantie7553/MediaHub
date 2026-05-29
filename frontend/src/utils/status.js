/**
 * Helper function for translating status into color
 * @param status the status we are matching the color to
 * @returns the class representation of the color we want displayed
 */
export function statusBadge(status) {
    switch(status) {
        case "approved": return "badge-success"
        case "pending": return "badge-warning"
        case "rejected": return "badge-error"
        default: return "badge-neutral"
    }
}

/**
 * Helper function for translating status into color
 * @param status the status we are matching the color to
 * @returns the class representation of the color we want displayed
 */
export function statusColor(status) {
    switch(status) {
        case "complete": return "status-success"
        case "downloading": return "status-info"
        case "queued": return "status-warning"
        case "failed": return "status-error"
        default: return "status-neutral"
    }
}

/**
 * Helper function for translating status into color
 * @param status the status we are matching the color to 
 * @returns the class representation of the color we want displayed
 */
export function mediaStatusBadge(status) {
    switch(status) {
        case "completed": return "badge-success"
        case "watching": return "badge-info"
        case "plan_to_watch": return "badge-warning"
        case "dropped": return "badge-error"
        default: return "badge-neutral"
    }
}

export function mangaBadge(status) {
    switch(status) {
        case "ongoing": return "badge-info"
        case "completed": return "badge-success"
        case "hiatus": return "badge-warning"
    }
}

export function animeBadge(status) {
    switch(status) {
        case "airing": return "badge-info"
        case "finished": return "badge-success"
        case "upcoming": return "badge-warning"
    }
}