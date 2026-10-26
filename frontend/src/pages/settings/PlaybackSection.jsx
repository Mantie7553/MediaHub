import { useEffect, useState } from "react";
import { MonitorPlay } from "lucide-react";
import api from "../../services/api";
import { toast } from "../../stores/useToastStore";
import { Panel } from "../../components/DataTable";

/**
 * How videos play for the user: whether openings, endings and recaps are skipped without asking.
 * The languages are chosen in the player, so they're sent back as they are
 */
export default function PlaybackSection() {
    // { audio_language, subtitle_language, auto_skip } once loaded
    const [prefs, setPrefs] = useState(null);
    const [saving, setSaving] = useState(false);

    useEffect(() => {
        api.get("/me/preferences")
            .then(res => setPrefs(res.data))
            .catch(err => toast.error(err, "Couldn't load your playback settings"));
    }, []);

    function toggleAutoSkip(on) {
        setSaving(true);
        api.put("/me/preferences", { ...prefs, auto_skip: on })
            .then(res => setPrefs(res.data))
            .catch(err => toast.error(err, "Couldn't save your playback settings"))
            .finally(() => setSaving(false));
    }

    return <Panel title="Playback" icon={MonitorPlay}>
        <label className="flex items-start justify-between gap-4 cursor-pointer">
            <span className="min-w-0">
                <span className="block font-medium">Skip openings and endings</span>
                <span className="block text-sm text-base-content/55">
                    Jumps past openings, endings, recaps and previews by itself. Otherwise the player shows a skip button.
                </span>
            </span>
            {prefs ? <input type="checkbox" className="toggle toggle-primary shrink-0 mt-0.5" checked={!!prefs.auto_skip}
                disabled={saving} onChange={e => toggleAutoSkip(e.target.checked)}/>
                : <span className="skeleton h-6 w-12 shrink-0 rounded-full"></span>}
        </label>
    </Panel>
}
