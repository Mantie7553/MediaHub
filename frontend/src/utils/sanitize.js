import DOMPurify from "dompurify"

// markup from outside (EPUB files, AniList descriptions) may format text but nothing more:
// no scripts or event handlers (DOMPurify removes those), no stylesheets that could restyle the
// app, and nothing that could pose as part of MediaHub's own interface, like forms or frames
const options = {
    FORBID_TAGS: ["style", "link", "form", "input", "button", "textarea", "select", "iframe", "frame", "object", "embed", "meta", "base"],
    FORBID_ATTR: ["formaction", "srcset"],
}

/**
 * HTML from outside MediaHub made safe to insert into the page
 * @param {string} html the markup
 * @returns the sanitized markup
 */
export function sanitizeHtml(html) {
    return DOMPurify.sanitize(html ?? "", options)
}
