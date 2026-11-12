import { useNavigate } from "react-router-dom"
import { House } from "lucide-react"
import SegmentedTabs from "../SegmentedTabs"
import useHomePage from "../../hooks/useHomePage"
import { homePages } from "../../utils/homePages"

/**
 * The tabs across the top of every home page: the overview and one per type. The user's home is
 * marked with a small house
 * @param {string} current the key of the page showing
 */
export default function HomeTabs({ current }) {
    const navigate = useNavigate()
    const { home, loading } = useHomePage()

    const options = homePages.map(({ key, label, icon }) => ({
        key,
        icon,
        label: <>
            {label}
            {!loading && home.key === key && <House size={12} className="opacity-60" aria-label="(your home)"/>}
        </>,
    }))

    return <SegmentedTabs options={options} value={current} onChange={key => navigate(homePages.find(p => p.key === key).path)}
        label="Home pages"/>
}
