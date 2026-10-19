import { useEffect, useRef, useState } from "react"
import { ChevronLeft, ChevronRight } from "lucide-react"

/**
 * A horizontally scrolling row of cards, with arrow buttons on desktop and faded edges
 * where there is more to scroll to
 * @param {any} children the <li> cards
 */
export default function Shelf({ children }) {
    const ref = useRef(null)
    const [edges, setEdges] = useState({ start: true, end: true })

    function update() {
        const el = ref.current
        if (!el) return
        setEdges({
            start: el.scrollLeft <= 4,
            end: el.scrollLeft + el.clientWidth >= el.scrollWidth - 4,
        })
    }

    useEffect(() => {
        update()
        const el = ref.current
        if (!el) return
        const observer = new ResizeObserver(update)
        observer.observe(el)
        return () => observer.disconnect()
    }, [children])

    function scroll(direction) {
        const el = ref.current
        el?.scrollBy({ left: direction * el.clientWidth * 0.85, behavior: "smooth" })
    }

    // fade only the edges that have more content past them
    const mask = `linear-gradient(to right, ${edges.start ? "#000" : "transparent"}, #000 3rem, #000 calc(100% - 3rem), ${edges.end ? "#000" : "transparent"})`

    return <div className="group/shelf relative -mx-4 sm:-mx-6 lg:-mx-1">
        <ul
            ref={ref}
            onScroll={update}
            className="shelf flex gap-3 sm:gap-4 overflow-x-auto flex-nowrap px-4 sm:px-6 lg:px-1 scroll-px-4 sm:scroll-px-6 lg:scroll-px-1 pt-1 pb-3"
            style={{ maskImage: mask, WebkitMaskImage: mask }}
        >
            {children}
        </ul>
        <ShelfArrow side="left" hidden={edges.start} onClick={() => scroll(-1)}/>
        <ShelfArrow side="right" hidden={edges.end} onClick={() => scroll(1)}/>
    </div>
}

function ShelfArrow({ side, hidden, onClick }) {
    const Icon = side === "left" ? ChevronLeft : ChevronRight
    return <button
        type="button"
        tabIndex={-1}
        aria-hidden="true"
        onClick={onClick}
        className={`hidden md:grid absolute top-[38%] -translate-y-1/2 ${side === "left" ? "left-2 lg:-left-3" : "right-2 lg:-right-3"} z-20 size-10 place-items-center rounded-full glass text-base-content shadow-raised
            transition-all duration-200 hover:bg-primary hover:text-primary-content
            ${hidden ? "opacity-0 pointer-events-none" : "opacity-0 group-hover/shelf:opacity-100"}`}
    >
        <Icon size={20}/>
    </button>
}
