import { useEffect, useRef } from "react";
import RatingModal from "./modals/RatingModal";
import useRatingStore from "../stores/useRatingStore";
import { saveListEntry } from "../services/mediaService";
import { toast } from "../stores/useToastStore";

/**
 * The rating dialog, shown whenever something calls rateTitle (stores/useRatingStore.js).
 * Rendered once, in the app's layout
 */
export default function RatingPrompt() {
    const request = useRatingStore(state => state.request);
    const dialogRef = useRef(null);

    useEffect(() => {
        const dialog = dialogRef.current;
        if (request && dialog && !dialog.open) dialog.showModal();
    }, [request]);

    function save(rating) {
        return saveListEntry({ id: request.entryId }, null, { rating })
            .then(() => { toast.success(rating == null ? "Rating cleared" : "Rating saved"); request.onSaved?.(); })
            .catch(err => { toast.error(err, "Could not save your rating"); throw err; });
    }

    return <RatingModal
        dialogRef={dialogRef}
        title={request?.title}
        rating={request?.rating ?? null}
        prompt={!!request?.prompt}
        onSave={save}
        onDismiss={() => request?.onDismiss?.()}
    />;
}
