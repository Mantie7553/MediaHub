import { useEffect, useState } from "react";

export default function MusicRequestModal({dialogRef, track, onConfirm}) {
    const [artist, setArtist] = useState("");
    const [album, setAlbum] = useState("");

    useEffect(() => {
        setArtist(track?.uploader ?? "");
        setAlbum("");
    }, [track]);

    function handleConfirm() {
        onConfirm({ artist, album });
        dialogRef.current.close();
    }

    return <dialog className="modal" ref={dialogRef} onClick={e => { e.stopPropagation(); e.preventDefault(); }}>
        <div className="modal-box flex flex-col gap-4">
            <h2 className="font-bold text-lg">{track?.title}</h2>
            <section className="flex flex-col gap-2">
                <label className="text-sm font-medium">Artist</label>
                <input
                    className="input input-bordered w-full"
                    value={artist}
                    onChange={e => setArtist(e.target.value)}
                />
            </section>
            <section className="flex flex-col gap-2">
                <label className="text-sm font-medium">Album</label>
                <input
                    className="input input-bordered w-full"
                    placeholder="Singles"
                    value={album}
                    onChange={e => setAlbum(e.target.value)}
                />
            </section>
            <section className="flex gap-2 justify-end">
                <button className="btn btn-secondary" onClick={() => dialogRef.current.close()}>Cancel</button>
                <button className="btn btn-primary" onClick={handleConfirm}>Confirm</button>
            </section>
        </div>
    </dialog>
}