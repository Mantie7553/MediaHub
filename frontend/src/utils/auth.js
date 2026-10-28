import { endSession, getRole as sessionRole } from "../services/session";
import useAudioStore from "../stores/useAudioStore";

/**
 * Role of the logged in user, read from the access token
 * @returns "admin", "user", or null when not logged in
 */
export function getRole() {
    return sessionRole();
}

/**
 * Whether the logged in user is an admin
 */
export function userIsAdmin() {
    return getRole() === "admin";
}

/**
 * Sign out: end the session on the server, forget it here and in other tabs, and go to the
 * login page
 * @param {function} navigate from useNavigate
 * @param {boolean} everywhere sign out on every device too
 */
export function logout(navigate, everywhere = false) {
    endSession(everywhere);
    // the next person to sign in here doesn't get this person's queue back
    useAudioStore.getState().stop();
    navigate("/login");
}
