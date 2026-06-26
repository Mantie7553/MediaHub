
/**
 * Validates a password against the application's password rules
 * @param {string} password the password to validate
 * @returns an error string if invalid, null if valid
 */
export function validatePassword(password) {
    if (password.length < 12) return "Password must be at least 12 characters";
    if (!/[A-Z]/.test(password)) return "Password must contain at least one uppercase letter";
    if (!/[a-z]/.test(password)) return "Password must contain at least one lowercase letter";
    if (!/[0-9]/.test(password)) return "Password must contain at least one number";
    if (!/[^A-Za-z0-9]/.test(password)) return "Password must contain at least one special character";
    return null;
}