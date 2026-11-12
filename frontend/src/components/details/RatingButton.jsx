import { Star } from "lucide-react";

/**
 * The user's rating in a details page's actions, opening the rating dialog
 * @param {number} rating 1 to 10, null when not rated
 * @param {function} onClick opens the rating dialog
 */
export default function RatingButton({ rating, onClick }) {
    const rated = rating != null;
    return <button
        type="button"
        className="btn border-white/10 bg-white/10 hover:bg-white/15 backdrop-blur gap-1.5 font-medium px-3"
        onClick={onClick}
        aria-label={rated ? `Your rating: ${rating} out of 10. Change it` : "Rate it"}
    >
        <Star size={16} className={rated ? "text-primary" : ""} fill={rated ? "currentColor" : "none"}/>
        {rated
            ? <span className="tabular-nums">{rating}<span className="text-base-content/50">/10</span></span>
            : <span className="hidden sm:inline">Rate</span>}
    </button>
}
