import { Pause, Play, Repeat, Repeat1, Shuffle } from "lucide-react"
import useAudioStore, { REPEAT_LABELS } from "../../stores/useAudioStore"

/**
 * An album or playlist's Play button. While its music is playing it pauses and resumes instead
 * of starting over, so the page works as the player's controls too
 * @param {object[]} tracks the queue to play, { id, title, artist, thumbnail }
 * @param {string} source this page's path, which the player bar links back to
 */
export function PlayButton({ tracks, source }) {
    const { isPlaying, playAll, pause, resume } = useAudioStore()
    const playingHere = useAudioStore(state => state.source === source && state.currentTrack !== null)

    function handleClick() {
        if (!playingHere) playAll(tracks, source)
        else if (isPlaying) pause()
        else resume()
    }

    const pausable = playingHere && isPlaying
    return <button className="btn btn-primary btn-lg shadow-glow px-6" onClick={handleClick} disabled={tracks.length === 0}>
        {pausable ? <Pause size={18} fill="currentColor"/> : <Play size={18} fill="currentColor"/>}
        {pausable ? "Pause" : playingHere ? "Resume" : "Play"}
    </button>
}

/**
 * Shuffle and repeat, the same switches as the player bar's (which has no room for them on
 * phones). Set before playing, they apply to the next Play
 */
export function QueueModeButtons() {
    const { shuffle, repeat, toggleShuffle, cycleRepeat } = useAudioStore()
    const RepeatIcon = repeat === "one" ? Repeat1 : Repeat
    const button = active => `btn btn-square border-white/10 bg-white/10 hover:bg-white/15 ${active ? "text-primary" : ""}`

    return <>
        <button className={button(shuffle)} onClick={toggleShuffle} aria-label={shuffle ? "Turn shuffle off" : "Shuffle"} aria-pressed={shuffle}>
            <Shuffle size={16}/>
        </button>
        <button className={button(repeat !== "off")} onClick={cycleRepeat} aria-label={REPEAT_LABELS[repeat]} aria-pressed={repeat !== "off"}>
            <RepeatIcon size={16}/>
        </button>
    </>
}
