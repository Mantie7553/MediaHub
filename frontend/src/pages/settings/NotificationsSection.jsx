import { useEffect, useState } from "react";
import { Bell } from "lucide-react";
import { toast } from "../../stores/useToastStore";
import { Panel } from "../../components/DataTable";
import { getNotificationPrefs, setNotificationPrefs } from "../../services/notificationsService";
import { settingsFor } from "../../utils/notifications";

/**
 * Which kinds of notification the user gets, one switch each. Admins also see the admin kinds
 */
export default function NotificationsSection() {
    // kind → on, once loaded
    const [prefs, setPrefs] = useState(null);
    const [saving, setSaving] = useState(null);

    useEffect(() => {
        getNotificationPrefs()
            .then(res => setPrefs(res.data))
            .catch(err => toast.error(err, "Couldn't load your notification settings"));
    }, []);

    function toggle(kind, on) {
        setSaving(kind);
        setPrefs(p => ({ ...p, [kind]: on }));
        setNotificationPrefs({ [kind]: on })
            .then(res => setPrefs(res.data))
            .catch(err => {
                setPrefs(p => ({ ...p, [kind]: !on }));
                toast.error(err, "Couldn't save your notification settings");
            })
            .finally(() => setSaving(null));
    }

    return <Panel title="Notifications" icon={Bell}>
        {prefs ? <ul className="flex flex-col gap-4">
            {settingsFor(prefs).map(s => <li key={s.kind}>
                <label className="flex items-start justify-between gap-4 cursor-pointer">
                    <span className="min-w-0">
                        <span className="block font-medium">{s.title}</span>
                        <span className="block text-sm text-base-content/55">{s.hint}</span>
                    </span>
                    <input type="checkbox" className="toggle toggle-primary shrink-0 mt-0.5" checked={!!prefs[s.kind]}
                        disabled={saving === s.kind} onChange={e => toggle(s.kind, e.target.checked)}/>
                </label>
            </li>)}
        </ul> : <div className="flex flex-col gap-4">
            {[0, 1, 2, 3].map(i => <div key={i} className="flex items-start justify-between gap-4">
                <span className="skeleton h-10 w-full max-w-xs"></span>
                <span className="skeleton h-6 w-12 shrink-0 rounded-full"></span>
            </div>)}
        </div>}
    </Panel>
}
