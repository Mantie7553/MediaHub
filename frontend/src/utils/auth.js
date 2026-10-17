
/**
 * Role of the logged in user, read from the access token
 * @returns "admin", "user", or null when not logged in
 */
export function getRole() {
    const token = localStorage.getItem("token");
    if (!token) return null;
    try {
        return JSON.parse(atob(token.split(".")[1])).role;
    } catch {
        return null;
    }
}
