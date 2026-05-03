import { useState } from "react"
import { useNavigate } from "react-router-dom"
import api from "../../services/api"

export default function Login() {
    const naviagte = useNavigate();
    const [username, setUserName] = useState("");
    const [email, setEmail] = useState("");
    const [password, setPassword] = useState("");
    const [confirmPass, setConfirmPass] = useState("");
    const [error, setError] = useState(null);
    const [signUp, setSignUp] = useState(false);

    function handleLogin(e) {
        e.preventDefault()
        api.post("/auth/login", {email, password})
        .then(res => {
            localStorage.setItem("token", res.data.token);
            naviagte("/");
        })
        .catch(err => {
            setError(err.message ?? "Invalid email or password");
        })
    }

    function handleSignUp(e) {
        e.preventDefault()

        if (!confirmPass || confirmPass != password) {
            setError("Passwords must match");
            return;
        }

        api.post("/auth/register", {username, email, password})
        .then(res => {
            localStorage.setItem("token", res.data.token);
            naviagte("/");
        })
        .catch(err => {
            setError(err.message ?? "Invalid email or password");
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
            <button type="button" className="link" onClick={() => setSignUp(!signUp)}>{signUp ? "Login" : "Sign Up"}</button>
        </form>
    </div>
    </div>
}