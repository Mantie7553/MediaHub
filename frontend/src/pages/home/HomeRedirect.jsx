import { Navigate } from "react-router-dom"
import useHomePage from "../../hooks/useHomePage"
import { CardRowSkeleton } from "../../components/states/Skeletons"

/**
 * "/" opens the page the user picked as home. If the setting can't load, the overview
 */
export default function HomeRedirect() {
    const { home, loading } = useHomePage()

    if (loading) return <div className="flex flex-col gap-10">
        <span className="skeleton h-10 w-64 rounded-field"></span>
        <CardRowSkeleton/>
        <CardRowSkeleton/>
    </div>

    return <Navigate to={home.path} replace/>
}
