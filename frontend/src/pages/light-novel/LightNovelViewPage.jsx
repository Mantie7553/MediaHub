import { NavLink, useParams } from "react-router-dom"
import Loading from "../../components/states/Loading"
import Error from "../../components/states/Error"
import { useMediaItem } from "../../hooks"

export default function LightNovelViewPage() {
    const { id } = useParams()
    const { item: ln, loading, error } = useMediaItem(id)

    if (loading) return <Loading />
    if (error) return <Error error={error} />
    if (!ln) return null

    return <div className="flex flex-col">
        <div className="flex gap-6">
            <img src={ln.cover_image_url} className="w-48 h-64 object-contain rounded-md" />
            <div className="flex flex-col gap-3">
                <h2 className="text-2xl font-bold">{ln.title}</h2>
                {ln.metadata.author && <span className="text-sm text-neutral-content">by {ln.metadata.author}</span>}
                <div className="flex flex-wrap gap-1">
                    {(ln.metadata.genres ?? []).map((genre, i) => (
                        <span key={i} className="badge">{genre}</span>
                    ))}
                </div>
            </div>
        </div>

        <h3 className="font-bold text-lg mt-6">Volumes</h3>
        <ul className="list">
            {(ln.metadata.volumes ?? []).map(volume => (
                <li key={volume.id} className="list-item hover:bg-base-300 transition-colors px-2 py-1">
                    <NavLink to={`/light-novels/${id}/volumes/${volume.id}/read`} className="block w-full">
                        {volume.title ?? `Volume ${volume.volume_number}`}
                    </NavLink>
                </li>
            ))}
        </ul>
    </div>
}