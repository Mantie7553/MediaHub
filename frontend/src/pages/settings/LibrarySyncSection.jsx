import { useState, useEffect, useRef } from "react";
import api from "../../services/api";
import Format from "../../utils/format";

const SOURCE_LABELS = {
    "sonarr": "TV (Sonarr)",
    "radarr": "Movies (Radarr)",
    "manga": "Manga",
    "light-novels": "Light Novels",
    "music": "Music",
};

const POLL_MS = 3000;

/**
 * Admin section that runs a full library sync (Sonarr, Radarr, manga, light novels, music)
 * and shows the progress and results of the current or last run
 */
export default function LibrarySyncSection() {
    const [status, setStatus] = useState(null);
    const [error, setError] = useState("");
    const pollRef = useRef(null);

    function fetchStatus() {
        return api.get("/admin/sync")
            .then(res => {
                setStatus(res.data);
                return res.data;
            })
            .catch(err => setError(err.response?.data ?? err.message));
    }

    // poll while a sync is running, stop once it's done
    function poll() {
        clearTimeout(pollRef.current);
        fetchStatus().then(data => {
            if (data?.running) pollRef.current = setTimeout(poll, POLL_MS);
        });
    }

    useEffect(() => {
        poll();
        return () => clearTimeout(pollRef.current);
    }, []);

    function handleStart() {
        setError("");
        api.post("/admin/sync")
            .then(res => {
                setStatus(res.data);
                pollRef.current = setTimeout(poll, POLL_MS);
            })
            .catch(err => {
                // 409: already running, just show its progress
                if (err.response?.status === 409) poll();
                else setError(err.response?.data ?? err.message);
            });
    }

    const running = status?.running;
    const results = status?.results ?? [];
    const done = new Set(results.map(r => r.source));

    return (
        <section className="flex flex-col gap-4">
            <div className="flex items-center justify-between">
                <h2 className="font-bold">Library Sync</h2>
                <button className="btn btn-primary btn-sm" onClick={handleStart} disabled={!status || running}>
                    {running && <span className="loading loading-spinner loading-xs" />}
                    {running ? "Syncing…" : "Sync Library"}
                </button>
            </div>
            <p className="text-sm opacity-70">
                Adds anything new from Sonarr, Radarr and the Manga, Light Novels and Music folders.
                {status?.next_run
                    ? ` Runs automatically every night, next on ${Format.dateTime(status.next_run)}.`
                    : " The nightly sync is turned off."}
            </p>
            {error && <p className="text-error text-sm">{error}</p>}

            {status?.started_at && (
                <div className="card overflow-x-auto">
                    <table className="table">
                        <thead>
                            <tr>
                                <th>Source</th>
                                <th>Result</th>
                                <th>Time</th>
                            </tr>
                        </thead>
                        <tbody>
                            {Object.keys(SOURCE_LABELS).map(source => {
                                const result = results.find(r => r.source === source);
                                return (
                                    <tr key={source}>
                                        <td>{SOURCE_LABELS[source]}</td>
                                        <td>
                                            {result ? (
                                                result.ok
                                                    ? <span className="badge badge-sm badge-success">{result.detail || "done"}</span>
                                                    : <span className="text-error text-sm">{result.error}</span>
                                            ) : running && !done.has(source) ? (
                                                <span className="badge badge-sm badge-ghost">waiting</span>
                                            ) : (
                                                <span className="opacity-50">not run</span>
                                            )}
                                        </td>
                                        <td>{result?.duration ?? ""}</td>
                                    </tr>
                                );
                            })}
                        </tbody>
                    </table>
                    <p className="text-xs opacity-50 px-4 pb-3">
                        {running ? "Started" : "Last run"} {Format.dateTime(status.started_at)}
                        {status.trigger === "schedule" ? " (nightly)" : " (manual)"}
                    </p>
                </div>
            )}
        </section>
    );
}
