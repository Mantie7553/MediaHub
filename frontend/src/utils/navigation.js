/**
 * Go to the previous page, or home when this page was opened directly (new tab, bookmark)
 * @param {function} navigate from useNavigate
 */
export function goBack(navigate) {
    if (window.history.state?.idx > 0) navigate(-1)
    else navigate("/")
}
