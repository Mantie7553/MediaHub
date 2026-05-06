
export default function Error({error}) {
    if (error) return <div className="alert alert-error">{error}</div>
}