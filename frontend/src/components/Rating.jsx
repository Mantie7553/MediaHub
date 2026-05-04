
export default function Rating({selected, id}) {
    return <div className="rating rating-xs">
        <div className="mask mask-star-2" aria-label="1 star" aria-current={selected === 1}></div>
        <div className="mask mask-star-2" aria-label="2 star" aria-current={selected === 2}></div>
        <div className="mask mask-star-2" aria-label="3 star" aria-current={selected === 3}></div>
        <div className="mask mask-star-2" aria-label="4 star" aria-current={selected === 4}></div>
        <div className="mask mask-star-2" aria-label="5 star" aria-current={selected === 5}></div>
    </div>
}