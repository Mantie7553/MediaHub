import { Component } from "react"
import { AlertTriangle, RotateCw } from "lucide-react"
import { useLocation } from "react-router-dom"
import { StateIcon } from "./EmptyState"

/**
 * Catches a crash while rendering a page and shows a message instead of a blank screen.
 * Give it key={location.pathname} so moving to another page clears it
 */
export default class ErrorBoundary extends Component {
    constructor(props) {
        super(props)
        this.state = { failed: false }
    }

    static getDerivedStateFromError() {
        return { failed: true }
    }

    componentDidCatch(error, info) {
        console.error("Page crashed:", error, info.componentStack)
    }

    render() {
        if (!this.state.failed) return this.props.children

        return <div role="alert" className="flex flex-col items-center text-center gap-3 py-20 px-4 animate-fade-up">
            <StateIcon icon={AlertTriangle} tone="error"/>
            <h2 className="text-xl font-semibold mt-2">Something broke on this page</h2>
            <p className="text-base-content/60 max-w-md">
                Reloading usually fixes it. If it keeps happening, the page has a bug.
            </p>
            <div className="flex gap-2 mt-3">
                <a href="/" className="btn btn-ghost">Home</a>
                <button className="btn btn-primary" onClick={() => window.location.reload()}>
                    <RotateCw size={16}/> Reload
                </button>
            </div>
        </div>
    }
}

/**
 * ErrorBoundary that resets when the user moves to another page
 */
export function PageBoundary({ children }) {
    const location = useLocation()
    return <ErrorBoundary key={location.pathname}>{children}</ErrorBoundary>
}
