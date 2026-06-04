import { useState, useEffect } from "react"
import { useParams, NavLink } from "react-router-dom"
import api from "../../services/api"
import Loading from "../../components/states/Loading"
import Error from "../../components/states/Error"

export default function LightNovelReader() {
    const { id, volumeId } = useParams()
    const [content, setContent] = useState(null)
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState(null)

    useEffect(() => {
        setLoading(true)
        api.get(`/light-novels/${id}/volumes/${volumeId}/content`)
            .then(res => setContent(res.data))
            .catch(err => setError(err.message))
            .finally(() => setLoading(false))
    }, [volumeId])

    if (loading) return <Loading />
    if (error) return <Error error={error} />

    return <div className="flex flex-col items-center pb-16">
        <div className="w-full max-w-2xl px-6">
            <div
                className="prose prose-invert max-w-none"
                dangerouslySetInnerHTML={{ __html: content }}
            />
        </div>
        <div className="fixed bottom-0 left-0 right-0 flex justify-center gap-4 p-4 bg-base-200 border-t border-base-300">
            <NavLink to={`/light-novels/${id}`} className="btn btn-sm">Back to Volumes</NavLink>
        </div>
    </div>
}