import { Navigate } from 'react-router-dom'

/**
 * Wrapper to lock users out of a page if they have not logged in
 * @param {any} children
 * @returns
 */
export default function ProtectedRoute({ children }) {
    const token = localStorage.getItem('token')
    if (!token) return <Navigate to="/login" />
    return children
}