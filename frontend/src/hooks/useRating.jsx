import { useEffect, useRef } from "react";
import RatingButton from "../components/details/RatingButton";
import { rateTitle } from "../stores/useRatingStore";
import { readStored, writeStored } from "../utils/storage";
import { dismissKey, shouldPromptRating } from "../utils/ratingPrompt";

/**
 * Rating a title from its details page: a button showing the user's rating that opens the rating
 * dialog, and the same dialog opening by itself when the title becomes completed without a rating
 * (on this page, or recently somewhere else like the player). The dialog is the app's one
 * RatingPrompt
 * @param {object} entry the user's list entry, when the title is on their list
 * @param {boolean} loaded the user's list has loaded, so a missing entry means "not on the list"
 * @param {string} title the title, for the dialog
 * @param {function} onSaved reloads the user's list after a rating is saved
 * @returns button, null when the title isn't on the list
 */
export default function useRating(entry, loaded, title, onSaved) {
    // the status last seen, undefined until the list has loaded
    const lastStatus = useRef(undefined);
    // the latest of everything the dialog needs, for the effect below without rerunning it
    const latest = useRef({ title, onSaved });
    useEffect(() => { latest.current = { title, onSaved }; });

    // the list can load before the title does; asking waits for both
    const ready = loaded && !!title;
    useEffect(() => {
        if (!ready) return;
        const previous = lastStatus.current;
        lastStatus.current = entry?.status ?? null;
        const dismissed = !!entry && readStored(dismissKey(entry), "") === "1";
        if (shouldPromptRating(previous, entry, Date.now(), dismissed)) {
            rateTitle({
                entryId: entry.id,
                title: latest.current.title,
                rating: null,
                prompt: true,
                onSaved: latest.current.onSaved,
                // closing it without saving is a "Not now" for this completion
                onDismiss: () => writeStored(dismissKey(entry), "1"),
            });
        }
    }, [ready, entry]);

    if (!entry) return { button: null };
    return {
        button: <RatingButton rating={entry.rating ?? null} onClick={() => rateTitle({
            entryId: entry.id,
            title,
            rating: entry.rating ?? null,
            prompt: false,
            onSaved,
        })}/>,
    };
}
