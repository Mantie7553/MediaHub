/**
 * Go to the previous page, or home when this page was opened directly (new tab, bookmark)
 * @param {function} navigate from useNavigate
 */
export function goBack(navigate) {
    if (window.history.state?.idx > 0) navigate(-1)
    else navigate("/")
}

/*
Viewers (player, manga and light novel readers) sit on top of a details page as one history
entry: details pages open them with openViewer, moving to the next episode/chapter/volume
replaces the entry (switchViewer), and leaving goes back to the details page underneath
(exitViewer). Pushing a new details page on the way out instead made Back on the details
page return to the viewer, and Back inside the viewer step through every chapter read.
*/

/**
 * Open a viewer from a details page
 * @param {function} navigate from useNavigate
 * @param {string} path the viewer's path
 * @param {object} state extra navigation state for the viewer (position, episode list)
 */
export function openViewer(navigate, path, state = {}) {
    navigate(path, { state: { ...state, fromDetails: true } })
}

/**
 * Move to another episode, chapter or volume without adding a history entry
 * @param {function} navigate from useNavigate
 * @param {object} location from useLocation, its state carries over
 * @param {string} path the viewer path to move to
 * @param {object} state navigation state to change (e.g. position)
 */
export function switchViewer(navigate, location, path, state = {}) {
    navigate(path, { replace: true, state: { ...location.state, ...state } })
}

/**
 * Leave a viewer for its details page
 * @param {function} navigate from useNavigate
 * @param {object} location from useLocation
 * @param {string|null} detailsPath where to go when the viewer wasn't opened from the details
 *  page (bookmark, reload). null goes back, or home
 */
export function exitViewer(navigate, location, detailsPath) {
    if (location.state?.fromDetails) navigate(-1)
    else if (detailsPath) navigate(detailsPath, { replace: true })
    else goBack(navigate)
}
