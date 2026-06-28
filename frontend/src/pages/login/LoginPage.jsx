import { useState } from "react"
import { useNavigate } from "react-router-dom"
import api from "../../services/api"
import { validatePassword } from "../../utils/validate";

/**
 * Login page layout
 * @returns
 */
export default function Login() {
    const navigate = useNavigate();
    const [username, setUserName] = useState("");
    const [email, setEmail] = useState("");
    const [password, setPassword] = useState("");
    const [confirmPass, setConfirmPass] = useState("");
    const [error, setError] = useState(null);
    const [signUp, setSignUp] = useState(false);

    /**
     * Login a user
     *      - make the request for the tokens
     *      - set token and refresh_token in localstorage for later use
     *      - send user to dashboard
     * @param {any} e the onClick event
     */
    function handleLogin(e) {
        e.preventDefault()

        if (!email || !password) { setError("Email and password are required"); return; }

        api.post("/auth/login", {email, password})
        .then(res => {
            localStorage.setItem("token", res.data.token);
            localStorage.setItem("refresh_token", res.data.refresh_token);
            navigate("/");
        })
        .catch(err => {
            setError(err.response?.data?.error ?? "Invalid email or password");
        })
    }

    /**
     * Sign up a user
     *      - confirm their password
     *      - make the request to add them to the database
     *      - send user to dashboard
     * @param {any} e the onClick event
     * @returns
     */
    function handleSignUp(e) {
        e.preventDefault()

        if (!username || !email || !password) { setError("All fields are required"); return; }

        const passwordError = validatePassword(password);
        if (passwordError) { setError(passwordError); return; }

        if (!confirmPass || confirmPass != password) {
            setError("Passwords must match");
            return;
        }
        
        api.post("/auth/register", {username, email, password})
        .then(res => {
            localStorage.setItem("token", res.data.token);
            localStorage.setItem("refresh_token", res.data.refresh_token);
            navigate("/");
        })
        .catch(err => {
            setError(err.response?.data?.error ?? "Something went wrong");
        })
    }

    return <div className="flex flex-col items-center justify-center min-h-screen gap-8">
        <h1 className="text-6xl">Media<span className="text-primary">Hub</span></h1>
        <div className="card card-border">
        <form onSubmit={signUp ? handleSignUp : handleLogin} className="card-body items-center">
            <h2 className="card-title">{signUp ? "Sign Up" : "Login"}</h2>
            {signUp && (
                <label className="input w-full">
                    <span className="label">Username</span>
                    <input value={username} onChange={e => setUserName(e.target.value)}/>
                </label>
            )}
            <label className="input w-full">
                <span className="label">Email</span>
                <input type="email" value={email} onChange={e => setEmail(e.target.value)}/>
            </label>
            <label className="input w-full">
                <span className="label">Password</span>
                <input type="password" value={password} onChange={e => setPassword(e.target.value)}/>
            </label>
            {signUp && (
                <label className="input w-full">
                    <span className="label">Confirm Password</span>
                    <input type="password" value={confirmPass} onChange={e => setConfirmPass(e.target.value)}/>
                </label>
            )}
            {error && <p className="text-error">{error}</p>}
            <button type="submit" className="btn btn-primary w-full">{signUp ? "Sign Up" : "Login"}</button>
            <div className="divider">OR</div>
            <button type="button" className="link" onClick={() => { setSignUp(!signUp); setError(null); }}>{signUp ? "Login" : "Sign Up"}</button>
        </form>
    </div>
    </div>
}