import { useState } from "react";
import { X } from "lucide-react";
import Autocomplete from "../Autocomplete";
import TextField from "../TextField";
import { albumFromTrack, artistFromTrack, canonical } from "../../utils/suggest";
import TrackByline, { ExplicitBadge } from "../TrackByline";

/**
 * Pick the artist and album a YouTube track is saved under before downloading it, and optionally
 * its track number. Artist and album suggest what the library already has: artists, then that
 * artist's albums. With no track number the server puts it after the album's last track. Enter confirms
 * @param {object} dialogRef ref for the <dialog>
 * @param {object} track the search result being requested
 * @param {object[]} artists the library's artists, { name, albums }
 * @param {function} onConfirm called with { artist, album, trackNumber }, artist and album spelled as
 *   the library has them, trackNumber a number or null
 */
export default function MusicRequestModal({ dialogRef, track, artists = [], onConfirm }) {
    return <dialog className="modal modal-bottom sm:modal-middle" ref={dialogRef}>
        <div className="modal-box flex flex-col gap-4">
            <button type="button" className="btn btn-sm btn-circle btn-ghost absolute right-3 top-3" aria-label="Close"
                onClick={() => dialogRef.current.close()}>
                <X size={16}/>
            </button>
            {/* keyed by track, so each request starts from that track's own artist */}
            {track && <RequestForm key={track.id} track={track} artists={artists}
                onCancel={() => dialogRef.current.close()}
                onConfirm={values => { onConfirm(values); dialogRef.current.close(); }}/>}
        </div>
    </dialog>
}

function RequestForm({ track, artists, onCancel, onConfirm }) {
    const [artist, setArtist] = useState(() => artistFromTrack(track));
    const [album, setAlbum] = useState(() => albumFromTrack(track));
    const [trackNumber, setTrackNumber] = useState("");

    const artistNames = artists.map(a => a.name);
    const known = artists.find(a => a.name === canonical(artistNames, artist));
    const albumNames = known?.albums ?? [];

    function handleSubmit(e) {
        e.preventDefault();
        if (!artist.trim()) return;
        onConfirm({
            artist: canonical(artistNames, artist),
            album: canonical(albumNames, album),
            trackNumber: trackNumber.trim() ? Number(trackNumber) : null,
        });
    }

    return <form className="flex flex-col gap-4" onSubmit={handleSubmit}>
        <div className="pr-8">
            <h3 className="text-xl font-semibold leading-snug">
                {track.title}
                {track.explicit && <span className="inline-block align-middle ml-2"><ExplicitBadge/></span>}
            </h3>
            <TrackByline track={track} className="text-sm text-base-content/60 mt-1"/>
        </div>
        <Autocomplete label="Artist" value={artist} onChange={setArtist} options={artistNames}
            placeholder="Who made it" required autoFocus
            hint="Tracks are saved in a folder per artist, then per album."/>
        <Autocomplete label="Album" value={album} onChange={setAlbum} options={albumNames}
            placeholder="Singles"
            hint={albumNames.length > 0 ? `${known.name} has ${albumNames.length} album${albumNames.length === 1 ? "" : "s"} in the library.` : "Leave empty to file it under Singles."}/>
        <TextField label="Track #" type="number" inputMode="numeric" min={1} max={999} step={1}
            value={trackNumber} onChange={e => setTrackNumber(e.target.value)} placeholder="Next"
            hint="Leave empty to add it after the album's last track."/>
        <div className="flex gap-2 justify-end">
            <button type="button" className="btn btn-ghost" onClick={onCancel}>Cancel</button>
            <button type="submit" className="btn btn-primary" disabled={!artist.trim()}>Download</button>
        </div>
    </form>
}
