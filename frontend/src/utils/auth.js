import { endSession, getRole as sessionRole } from "../services/session";

/**
 * Role of the logged in user, read from the access token
 * @returns "admin", "user", or null when not logged in
 */
export function getRole() {
    return sessionRole();
}

/**
 * Sign out: end the session on the server, forget it here and in other tabs, and go to the
 * login page
 * @param {function} navigate from useNavigate
 * @param {boolean} everywhere sign out on every device too
 */
export function logout(navigate, everywhere = false) {
    endSession(everywhere);
    navigate("/login");
}
