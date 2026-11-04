import { Maximize, Minimize, Pause, PictureInPicture2, Play, RotateCcw, RotateCw, SkipBack, SkipForward, Volume1, Volume2, VolumeX } from "lucide-react"
import Format from "../../utils/format"

const BUTTON = "btn btn-ghost btn-sm btn-square text-white hover:bg-white/10 border-0"

/**
 * The row under the seek bar. On phones play, the skips and the episode buttons move to the
 * middle of the screen (CenterControls) and this keeps the time and the right-hand buttons
 * @param {object} video what the video is doing, from useVideoState
 * @param {number} skipSecs how far the skip buttons go
 * @param {object} actions { togglePlay, skip(secs), prev, next, toggleMute, setVolume(0-1),
 *  toggleFullscreen, togglePictureInPicture }, prev/next null when there's no such episode
 * @param {boolean} hasEpisodes playing an episode of a list, so the episode buttons show
 * @param {boolean} fullscreen the player is fullscreen
 * @param {any} children more buttons before Fullscreen, like the settings menu
 */
export default function PlayerControls({ video, skipSecs, actions, hasEpisodes, fullscreen, children }) {
    const VolumeIcon = video.muted || video.volume === 0 ? VolumeX : video.volume < 0.5 ? Volume1 : Volume2

    return <div className="flex items-center gap-1 sm:gap-1.5">
        <div className="hidden sm:flex items-center gap-1">
            <button className={BUTTON} onClick={actions.togglePlay} aria-label={video.paused ? "Play" : "Pause"}>
                {video.paused ? <Play size={20} fill="currentColor"/> : <Pause size={20} fill="currentColor"/>}
            </button>
            <button className={BUTTON} onClick={() => actions.skip(-skipSecs)} aria-label={`Back ${skipSecs} seconds`}>
                <RotateCcw size={18}/>
            </button>
            <button className={BUTTON} onClick={() => actions.skip(skipSecs)} aria-label={`Forward ${skipSecs} seconds`}>
                <RotateCw size={18}/>
            </button>
            {hasEpisodes && <>
                <button className={BUTTON} disabled={!actions.prev} onClick={actions.prev ?? undefined} aria-label="Previous episode">
                    <SkipBack size={18}/>
                </button>
                <button className={BUTTON} disabled={!actions.next} onClick={actions.next ?? undefined} aria-label="Next episode">
                    <SkipForward size={18}/>
                </button>
            </>}
        </div>
        <span className="px-1.5 text-xs sm:text-sm tabular-nums text-white/80 whitespace-nowrap">
            {Format.duration(video.currentTime)} <span className="text-white/40">/ {Format.duration(video.duration)}</span>
        </span>
        <div className="flex-1"></div>
        <div className="group/volume hidden sm:flex pointer-coarse:hidden items-center">
            <input
                type="range"
                min={0}
                max={1}
                step={0.05}
                value={video.muted ? 0 : video.volume}
                onChange={e => actions.setVolume(Number(e.target.value))}
                className="range range-xs w-0 opacity-0 transition-all duration-200 group-hover/volume:w-20 group-hover/volume:opacity-100 focus-visible:w-20 focus-visible:opacity-100 text-primary"
                aria-label="Volume"
            />
            <button className={BUTTON} onClick={actions.toggleMute} aria-label={video.muted ? "Unmute" : "Mute"}>
                <VolumeIcon size={18}/>
            </button>
        </div>
        {children}
        {/* touch devices only: on a computer the small window has no use, and Chrome and Edge show no
            subtitles in it */}
        {typeof document !== "undefined" && document.pictureInPictureEnabled && (
            <button className={`${BUTTON} hidden pointer-coarse:inline-flex`} onClick={actions.togglePictureInPicture} aria-label="Picture in picture">
                <PictureInPicture2 size={18}/>
            </button>
        )}
        <button className={BUTTON} onClick={actions.toggleFullscreen} aria-label={fullscreen ? "Exit fullscreen" : "Fullscreen"}>
            {fullscreen ? <Minimize size={18}/> : <Maximize size={18}/>}
        </button>
    </div>
}

/**
 * Big play, skip and episode buttons in the middle of the screen, on phones only
 * @param {object} video what the video is doing, from useVideoState
 * @param {number} skipSecs how far the skip buttons go
 * @param {object} actions the same as PlayerControls'
 * @param {boolean} hasEpisodes playing an episode of a list, so the episode buttons show
 */
export function CenterControls({ video, skipSecs, actions, hasEpisodes }) {
    const round = "btn btn-circle border-0 bg-black/40 hover:bg-black/60 text-white backdrop-blur"
    return <div className="flex items-center gap-5">
        {hasEpisodes && <button className={`${round} btn-sm`} disabled={!actions.prev} onClick={actions.prev ?? undefined} aria-label="Previous episode">
            <SkipBack size={16}/>
        </button>}
        <button className={round} onClick={() => actions.skip(-skipSecs)} aria-label={`Back ${skipSecs} seconds`}>
            <RotateCcw size={22}/>
        </button>
        <button className={`${round} btn-lg size-16`} onClick={actions.togglePlay} aria-label={video.paused ? "Play" : "Pause"}>
            {video.paused ? <Play size={30} fill="currentColor" className="ml-1"/> : <Pause size={30} fill="currentColor"/>}
        </button>
        <button className={round} onClick={() => actions.skip(skipSecs)} aria-label={`Forward ${skipSecs} seconds`}>
            <RotateCw size={22}/>
        </button>
        {hasEpisodes && <button className={`${round} btn-sm`} disabled={!actions.next} onClick={actions.next ?? undefined} aria-label="Next episode">
            <SkipForward size={16}/>
        </button>}
    </div>
}
