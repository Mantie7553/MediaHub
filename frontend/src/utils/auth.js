import api from "../services/api";
import { clearCached } from "../services/apiCache";


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

/**
 * Sign out: revoke the refresh token on the server (best effort), forget both tokens and go to
 * the login page
 * @param {function} navigate from useNavigate
 */
export function logout(navigate) {
    api.delete("/auth/logout", {
        data: {refresh_token: localStorage.getItem("refresh_token")}
    });
    localStorage.removeItem("token");
    localStorage.removeItem("refresh_token");
    // the next person to sign in here mustn't see this one's lists
    clearCached();
    navigate("/login");
}
