/**
 * The title block at the top of a page
 * @param {string} title the page title
 * @param {any} subtitle a line under the title
 * @param {any} actions buttons shown on the right
 * @param {any} eyebrow a small label above the title
 */
export default function PageHeader({ title, subtitle, actions, eyebrow }) {
    return <header className="flex flex-wrap items-end justify-between gap-4 mb-8 animate-fade-up">
        <div className="min-w-0">
            {eyebrow && <p className="text-xs font-medium uppercase tracking-wider text-primary mb-1">{eyebrow}</p>}
            <h1 className="text-3xl lg:text-4xl font-semibold tracking-tight">{title}</h1>
            {subtitle && <p className="mt-1.5 text-base-content/60">{subtitle}</p>}
        </div>
        {actions && <div className="flex items-center gap-2">{actions}</div>}
    </header>
}
