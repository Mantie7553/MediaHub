import { useState } from "react";
import { House } from "lucide-react";
import { Panel } from "../../components/DataTable";
import { SelectField } from "../../components/TextField";
import useHomePage from "../../hooks/useHomePage";
import { homePages } from "../../utils/homePages";
import { toast } from "../../stores/useToastStore";

/**
 * Which page Home opens: the overview of everything the user follows, or one type's page for
 * someone who mostly uses MediaHub for one thing
 */
export default function HomePageSection() {
    const { home, loading, setHomePage } = useHomePage();
    const [saving, setSaving] = useState(false);

    function change(key) {
        setSaving(true);
        setHomePage(key)
            .then(() => toast.success("Home page saved"))
            .catch(err => toast.error(err, "Couldn't change your home page"))
            .finally(() => setSaving(false));
    }

    return <Panel title="Home page" icon={House}>
        <p className="text-sm text-base-content/55 mb-4">
            The page Home opens. Pick a type to always see it first, or the overview for everything you follow.
        </p>
        {loading ? <span className="skeleton h-11 w-full block rounded-field"></span> : (
            <SelectField label="Open on" value={home.key} disabled={saving} onChange={e => change(e.target.value)}>
                {homePages.map(page => <option key={page.key} value={page.key}>{page.label}</option>)}
            </SelectField>
        )}
    </Panel>
}
