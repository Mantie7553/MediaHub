import { useState } from "react"
import { useNavigate, useSearchParams } from "react-router-dom"
import api from "../../services/api"
import { setAccessToken } from "../../services/session";
import { errorMessage } from "../../utils/errors";
import { validatePassword } from "../../utils/validate";
import { AlertCircle, KeyRound, Mail, Ticket, User } from "lucide-react";
import Logo from "../../components/layout/Logo";
import TextField from "../../components/TextField";

/**
 * Login page layout
 * @returns
 */
export default function Login() {
    const navigate = useNavigate();
    // invite links look like /login?invite=XXXX-XXXX-XXXX-XXXX and open straight to sign up
    const [searchParams] = useSearchParams();
    const [inviteCode, setInviteCode] = useState(searchParams.get("invite") ?? "");
    const [username, setUserName] = useState("");
    const [email, setEmail] = useState("");
    const [password, setPassword] = useState("");
    const [confirmPass, setConfirmPass] = useState("");
    const [error, setError] = useState(null);
    const [signUp, setSignUp] = useState(searchParams.has("invite"));
    // a request is in flight, the button shows a spinner
    const [submitting, setSubmitting] = useState(false);

    /**
     * Login a user
     *      - make the request for the tokens: the access token comes back, the session is
     *        an HttpOnly cookie the browser keeps
     *      - send user to dashboard
     * @param {any} e the onClick event
     */
    function handleLogin(e) {
        e.preventDefault()

        if (!email || !password) { setError("Email and password are required"); return; }

        setSubmitting(true);
        api.post("/auth/login", {email, password})
        .then(res => {
            setAccessToken(res.data.token);
            navigate("/");
        })
        .catch(err => {
            setError(err.response?.status === 401 ? "Invalid email or password" : errorMessage(err, "Couldn't sign in"));
        })
        .finally(() => setSubmitting(false))
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

        if (!inviteCode || !username || !email || !password) { setError("All fields are required"); return; }

        const passwordError = validatePassword(password);
        if (passwordError) { setError(passwordError); return; }

        if (!confirmPass || confirmPass != password) {
            setError("Passwords must match");
            return;
        }
        
        setSubmitting(true);
        api.post("/auth/register", {username, email, password, invite_code: inviteCode})
        .then(res => {
            setAccessToken(res.data.token);
            navigate("/");
        })
        .catch(err => {
            setError(errorMessage(err, "Couldn't create the account"));
        })
        .finally(() => setSubmitting(false))
    }

    function switchMode(toSignUp) {
        setSignUp(toSignUp);
        setError(null);
    }

    return <div className="relative isolate min-h-dvh flex items-center justify-center px-4 py-10 pt-[max(2.5rem,env(safe-area-inset-top))] overflow-hidden">
        {/* slow drifting glows that mirror the logo: orange from the top left, purple along the bottom */}
        <div className="pointer-events-none absolute inset-0 -z-10 overflow-hidden" aria-hidden="true">
            <div className="absolute inset-0 bg-linear-135 from-primary/12 via-transparent to-secondary/18"></div>
            <div className="absolute -top-56 -left-48 size-[40rem] rounded-full bg-primary/35 blur-3xl animate-blob"></div>
            <div className="absolute top-[18%] left-[22%] size-[22rem] rounded-full bg-[#c2410c]/30 blur-3xl animate-blob-alt [animation-duration:26s]"></div>
            <div className="absolute -top-24 right-[8%] size-[18rem] rounded-full bg-accent/15 blur-3xl animate-blob-alt [animation-duration:30s] [animation-direction:reverse]"></div>
            <div className="absolute -bottom-56 -left-40 size-[34rem] rounded-full bg-secondary/35 blur-3xl animate-blob-alt [animation-duration:22s] [animation-delay:-7s]"></div>
            <div className="absolute -bottom-64 -right-48 size-[42rem] rounded-full bg-secondary/40 blur-3xl animate-blob [animation-duration:28s] [animation-delay:-12s]"></div>
            <div className="absolute top-[45%] -right-24 size-[20rem] rounded-full bg-[#a855f7]/20 blur-3xl animate-blob [animation-duration:20s] [animation-direction:reverse]"></div>
        </div>

        <div className="w-full max-w-md animate-fade-up">
            <div className="flex justify-center mb-8">
                <Logo size="lg"/>
            </div>

            <div className="surface-raised p-6 sm:p-8">
                <div role="tablist" aria-label="Sign in or sign up" className="grid grid-cols-2 gap-1 rounded-full bg-base-100/60 ring-1 ring-white/5 p-1">
                    {[{ key: false, label: "Log in" }, { key: true, label: "Sign up" }].map(tab => (
                        <button
                            key={tab.label}
                            type="button"
                            role="tab"
                            aria-selected={signUp === tab.key}
                            onClick={() => switchMode(tab.key)}
                            className={`h-9 rounded-full text-sm transition-all duration-200
                                ${signUp === tab.key ? "bg-primary text-primary-content font-medium shadow-glow" : "text-base-content/60 hover:text-base-content"}`}
                        >
                            {tab.label}
                        </button>
                    ))}
                </div>

                <h1 className="text-2xl font-semibold tracking-tight mt-7">{signUp ? "Create your account" : "Welcome back"}</h1>
                <p className="text-sm text-base-content/55 mt-1">
                    {signUp ? "You'll need an invite code from whoever runs this server." : "Log in to pick up where you left off."}
                </p>

                <form onSubmit={signUp ? handleSignUp : handleLogin} className="flex flex-col gap-4 mt-6" noValidate>
                    {signUp && <TextField label="Invite code" icon={Ticket} value={inviteCode} onChange={e => setInviteCode(e.target.value)}
                        placeholder="XXXX-XXXX-XXXX-XXXX" autoComplete="off" autoCapitalize="characters" spellCheck={false}/>}
                    {signUp && <TextField label="Username" icon={User} value={username} onChange={e => setUserName(e.target.value)}
                        autoComplete="username" autoCapitalize="none" spellCheck={false}/>}
                    <TextField label="Email" icon={Mail} type="email" value={email} onChange={e => setEmail(e.target.value)}
                        autoComplete="email" inputMode="email" autoCapitalize="none" spellCheck={false}/>
                    <TextField label="Password" icon={KeyRound} type="password" value={password} onChange={e => setPassword(e.target.value)}
                        autoComplete={signUp ? "new-password" : "current-password"}/>
                    {signUp && <TextField label="Confirm password" icon={KeyRound} type="password" value={confirmPass} onChange={e => setConfirmPass(e.target.value)}
                        autoComplete="new-password"/>}

                    {error && <div role="alert" className="flex items-start gap-2.5 rounded-field bg-error/10 ring-1 ring-error/25 px-3.5 py-3 text-sm text-error">
                        <AlertCircle size={17} className="shrink-0 mt-0.5"/> <span>{error}</span>
                    </div>}

                    <button type="submit" className="btn btn-primary btn-lg w-full shadow-glow mt-2" disabled={submitting}>
                        {submitting && <span className="loading loading-spinner loading-sm"/>}
                        {signUp ? "Create account" : "Log in"}
                    </button>
                </form>
            </div>

            <p className="text-center text-sm text-base-content/50 mt-6">
                {signUp ? "Already have an account? " : "Got an invite? "}
                <button type="button" className="font-medium text-primary hover:underline" onClick={() => switchMode(!signUp)}>
                    {signUp ? "Log in" : "Sign up"}
                </button>
            </p>
        </div>
    </div>
}
