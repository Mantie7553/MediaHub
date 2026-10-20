
/**
 * Validates a password against the application's password rules
 * @param {string} password the password to validate
 * @returns an error string if invalid, null if valid
 */
export function validatePassword(password) {
    if (password.length < 12) return "Password must be at least 12 characters";
    // the server's hash only uses the first 72 bytes, so it refuses longer passwords
    if (new TextEncoder().encode(password).length > 72) return "Password must be at most 72 bytes (about 72 letters)";
    if (!/[A-Z]/.test(password)) return "Password must contain at least one uppercase letter";
    if (!/[a-z]/.test(password)) return "Password must contain at least one lowercase letter";
    if (!/[0-9]/.test(password)) return "Password must contain at least one number";
    if (!/[^A-Za-z0-9]/.test(password)) return "Password must contain at least one special character";
    return null;
}