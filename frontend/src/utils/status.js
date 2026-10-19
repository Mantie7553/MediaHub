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
        case "current": return "badge-info"
        case "planned": return "badge-warning"
        case "dropped": return "badge-error"
        default: return "badge-neutral"
    }
}

/**
 * The dot color for a list status, used on the chips over cover art
 * @param status the status we are matching the color to
 * @returns the background class for the dot
 */
export function mediaStatusDot(status) {
    switch(status) {
        case "completed": return "bg-success"
        case "current": return "bg-info"
        case "planned": return "bg-warning"
        case "dropped": return "bg-error"
        case "wishlist": return "bg-secondary"
        default: return "bg-neutral-content"
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