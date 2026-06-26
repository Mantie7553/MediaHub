import { ChevronLeft, ChevronRight } from "lucide-react"

/**
 * A horizontal scroll picker for numeric values
 * @param {number} value - current value
 * @param {function} onChange - called with new value
 * @param {number} min - minimum value
 * @param {number} max - maximum value
 * @param {number} unsetValue - value that represents "not set"
 * @param {string} unsetLabel - label to display when at unset value
 */
export default function ScrollPicker({ value, onChange, min = 0, max = 10, unsetValue = 0, unsetLabel = "-" }) {
    function decrement() {
        if (value > min) onChange(value - 1);
    }

    function increment() {
        if (value < max) onChange(value + 1);
    }

    return (
        <div className="flex items-center gap-2 w-fit input border-base-300">
            <button
                type="button"
                className="btn btn-sm btn-ghost hover:text-primary"
                onClick={decrement}
                disabled={value <= min}
            >
                <ChevronLeft size={24} strokeWidth={4}/>
            </button>
            <span className="w-8 text-center font-bold text-lg">
                {value === unsetValue ? unsetLabel : value}
            </span>
            <button
                type="button"
                className="btn btn-sm btn-ghost hover:text-primary"
                onClick={increment}
                disabled={value >= max}
            >
                <ChevronRight size={24} strokeWidth={4}/>
            </button>
        </div>
    )
}