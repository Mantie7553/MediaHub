import { useState, useEffect } from "react"
import { useParams } from "react-router-dom"
import api from "../../services/api"
import Loading from "../../components/states/Loading"
import Error from "../../components/states/Error"

export default function PlayerPage() {
    const { id } = useParams()
    const [error, setError] = useState("")

    useEffect(() => {
        api.get(`/plex/stream/${id}`)
            .then(resp => window.location.href = resp.data.url)
            .catch(err => setError(err.message))
    }, [id])

    if (error) return <Error error={error} />

    return <Loading />
}