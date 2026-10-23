import { useState } from "react"
import { useNavigate } from "react-router-dom"
import useDocumentTitle from "../../hooks/useDocumentTitle"
import { ArrowLeft } from "lucide-react"
import Cover from "../Cover"
import { goBack } from "../../utils/navigation"
import { sanitizeHtml } from "../../utils/sanitize"

// descriptions longer than this get a Read more toggle
const LONG_DESCRIPTION = 320

/**
 * The top of a details page: the cover blurred behind everything, the poster, title, facts,
 * description and actions. Runs to the edges of the page
 * @param {object} cover { src, source } as for components/Cover.jsx
 * @param {boolean} square album art instead of a poster
 * @param {any} poster drawn in the poster's place instead of the cover, like a playlist's mosaic;
 *  the cover still tints the background
 * @param {string} eyebrow a small label above the title, like the media type
 * @param {string} title the title
 * @param {any[]} meta short facts shown in a line, empty ones are skipped
 * @param {any} badges chips beside the facts, like airing status
 * @param {string[]} genres shown as chips
 * @param {string} description plain text, or HTML with html set
 * @param {boolean} html the description is HTML from the metadata source
 * @param {any} primary the main button, full width on phones
 * @param {any} actions the other buttons, in a row under the main one on phones
 * @param {any} note a line under the actions, like why it can't be played
 */
export default function DetailsHero({ cover, square = false, poster, eyebrow, title, meta = [], badges, genres = [], description, html = false, primary, actions, note }) {
    const navigate = useNavigate()
    useDocumentTitle(title)
    const facts = meta.filter(Boolean)

    return <section className="relative z-10 isolate -mx-4 sm:-mx-6 lg:-mx-10 -mt-5 sm:-mt-6 lg:-mt-8 px-4 sm:px-6 lg:px-10 pt-3 lg:pt-6 pb-10 mb-2">
        {/* the cover, blurred into a wash of its colors. Clipped here and not on the section, so
            the action menus can open past the hero. Past the page's max width its edges fade out */}
        <div className="absolute inset-0 -z-10 overflow-hidden min-[1680px]:[mask-image:linear-gradient(to_right,transparent,#000_6rem,#000_calc(100%-6rem),transparent)]" aria-hidden="true">
            {cover?.src && <Cover src={cover.src} source={cover.source} className="size-full object-cover scale-125 blur-3xl saturate-150 opacity-60"/>}
            <div className="absolute inset-0 bg-linear-to-b from-base-100/20 via-base-100/75 to-base-100"></div>
            <div className="absolute inset-0 bg-linear-to-r from-base-100/70 via-transparent to-transparent"></div>
        </div>

        <button className="btn btn-ghost btn-sm -ml-2 text-base-content/80 hover:bg-white/10" onClick={() => goBack(navigate)}>
            <ArrowLeft size={18} strokeWidth={2}/> Back
        </button>

        <div className="mt-4 lg:mt-6 flex flex-col sm:flex-row items-center sm:items-end gap-6 lg:gap-10 animate-fade-up">
            <div className={`shrink-0 overflow-hidden rounded-box bg-base-300 shadow-raised ring-1 ring-white/10
                ${square ? "w-52 sm:w-56 lg:w-64 aspect-square" : "w-44 sm:w-52 lg:w-60 aspect-[2/3]"}`}>
                {poster ?? <Cover src={cover?.src} source={cover?.source} alt={title} className="size-full object-cover"/>}
            </div>

            <div className="flex flex-col gap-4 min-w-0 w-full items-center text-center sm:items-start sm:text-left">
                <div>
                    {eyebrow && <p className="text-xs font-medium uppercase tracking-wider text-primary mb-2">{eyebrow}</p>}
                    <h1 className="text-3xl sm:text-4xl lg:text-5xl font-semibold tracking-tight leading-tight text-balance">{title}</h1>
                </div>

                {(facts.length > 0 || badges) && <div className="flex flex-wrap items-center justify-center sm:justify-start gap-x-2 gap-y-1.5 text-sm text-base-content/70">
                    {badges}
                    {facts.map((fact, i) => <span key={i} className="flex items-center gap-2">
                        {(i > 0 || badges) && <span className="text-base-content/30" aria-hidden="true">•</span>}
                        {fact}
                    </span>)}
                </div>}

                {genres.length > 0 && <div className="flex flex-wrap justify-center sm:justify-start gap-1.5">
                    {genres.map(genre => <span key={genre} className="rounded-full bg-white/8 ring-1 ring-white/10 px-2.5 py-0.5 text-xs text-base-content/80">{genre}</span>)}
                </div>}

                {description && <Description text={description} html={html}/>}

                {(primary || actions) && <div className="w-full max-w-md sm:max-w-none flex flex-col sm:flex-row sm:flex-wrap sm:items-center gap-2 pt-1">
                    {primary && <div className="flex *:flex-1 sm:*:flex-none">{primary}</div>}
                    {actions && <div className="flex items-center gap-2">{actions}</div>}
                </div>}
                {note && <p className="text-sm text-base-content/50">{note}</p>}
            </div>
        </div>
    </section>
}

function Description({ text, html }) {
    const [expanded, setExpanded] = useState(false)
    const plainLength = html ? text.replace(/<[^>]+>/g, "").length : text.length
    const long = plainLength > LONG_DESCRIPTION

    const body = html
        ? <div dangerouslySetInnerHTML={{ __html: sanitizeHtml(text) }}/>
        : <div>{text}</div>

    return <div className="max-w-2xl text-sm leading-relaxed text-base-content/75 text-left">
        <div className={long && !expanded ? "line-clamp-4" : ""}>{body}</div>
        {long && <button className="mt-1 text-sm font-medium text-primary hover:underline" onClick={() => setExpanded(!expanded)}>
            {expanded ? "Show less" : "Read more"}
        </button>}
    </div>
}
