
export default function Loading({loading}) {
    if (loading) return <div className="flex justify-center p-10">
        <span className="loading loading-spinner loading-lg"></span>
    </div>
}