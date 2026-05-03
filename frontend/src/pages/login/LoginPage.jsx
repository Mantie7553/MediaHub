import { useState } from "react"
import { useNavigate } from "react-router-dom"
import api from "../../services/api"

export default function Login() {
    const naviagte = useNavigate();
    const [email, setEmail] = useState("");
    const [password, setPassword] = useState("");
    const [error, setError] = useState(null);

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

    return <div className="flex flex-col items-center justify-center min-h-screen gap-8">
        <h1 className="text-6xl">Media<span className="text-primary">Hub</span></h1>
        <div className="card card-border">
        <form onSubmit={handleLogin} className="card-body items-center">
            <h2 className="card-title">Login</h2>
            <label className="input">
                <span className="label">Email</span>
                <input type="email" value={email} onChange={e => setEmail(e.target.value)}/>
            </label>
            <label className="input">
                <span className="label">Password</span>
                <input type="password" value={password} onChange={e => setPassword(e.target.value)}/>
            </label>
            {error && <p className="text-error">{error}</p>}
            <button type="submit" className="btn btn-primary w-full">Login</button>
            <div className="divider">OR</div>
            <button className="link">Sign Up</button>
        </form>
    </div>
    </div>
}