import { create } from "zustand"
import { readStored, writeStored } from "../utils/storage"
import { shuffled } from "../utils/collections"

const STORAGE_KEY = "mediahub.player"
const REPEAT_MODES = ["off", "all", "one"]

// the queue's indexes in play order: in order, or shuffled with the track at `first` up front
function playOrder(length, shuffle, first) {
    const indexes = [...Array(length).keys()]
    if (!shuffle) return indexes
    return [first, ...shuffled(indexes.filter(i => i !== first))]
}

// the player as it was before the page was reloaded, paused where it was left
function restore() {
    try {
        const saved = JSON.parse(readStored(STORAGE_KEY, "null"))
        const { queue, order, position } = saved ?? {}
        if (!Array.isArray(queue) || !Array.isArray(order) || order.length !== queue.length || !queue[order[position]]) return {}
        return {
            queue, order, position,
            currentTrack: queue[order[position]],
            shuffle: saved.shuffle === true,
            repeat: REPEAT_MODES.includes(saved.repeat) ? saved.repeat : "off",
            resumeAt: Number(saved.time) || 0,
            source: typeof saved.source === "string" && saved.source.startsWith("/") ? saved.source : null,
        }
    } catch {
        return {}
    }
}

// source is the page the queue was played from (/albums/<id>, /playlists/<id>), which the bar links to
const empty = { currentTrack: null, isPlaying: false, queue: [], order: [], position: -1, resumeAt: 0, source: null }

const useAudioStore = create((set, get) => ({
    ...empty,
    shuffle: false,
    // off, all (the queue starts over after its last track) or one (the track plays again)
    repeat: "off",
    // where the restored track picks up, in seconds
    ...restore(),

    // play a single track, from the page given when it has one (its album)
    play: (track, source = null) => get().playAlbum([track], 0, source),

    // play from a list of tracks starting at an index, shuffled after it when shuffle is on.
    // source is the page they're played from
    playAlbum: (tracks, index, source = null) => {
        const order = playOrder(tracks.length, get().shuffle, index)
        set({ queue: tracks, order, position: order.indexOf(index), currentTrack: tracks[index], isPlaying: true, resumeAt: 0, source })
    },

    pause: () => set({ isPlaying: false }),
    resume: () => set({ isPlaying: true }),

    stop: () => {
        set(empty)
        writeStored(STORAGE_KEY, "null")
    },

    // the next track. After the last one the queue starts over: playing on with repeat all,
    // paused on the first track otherwise
    next: () => {
        const { queue, order, position, repeat } = get()
        if (queue.length === 0) return
        const atEnd = position >= order.length - 1
        const nextPosition = atEnd ? 0 : position + 1
        set({
            position: nextPosition,
            currentTrack: queue[order[nextPosition]],
            resumeAt: 0,
            ...(atEnd && repeat !== "all" ? { isPlaying: false } : {}),
        })
    },

    // the previous track, the last one from the first with repeat all, or the first track again
    prev: () => {
        const { queue, order, position, repeat } = get()
        if (queue.length === 0) return
        const prevPosition = position > 0 ? position - 1 : repeat === "all" ? order.length - 1 : 0
        set({ position: prevPosition, currentTrack: queue[order[prevPosition]], resumeAt: 0 })
    },

    // shuffling keeps the current track playing and shuffles the rest after it; turning it off
    // carries on in the queue's own order from the current track
    toggleShuffle: () => {
        const { queue, order, position, shuffle } = get()
        const current = order[position] ?? 0
        const nextOrder = playOrder(queue.length, !shuffle, current)
        set({ shuffle: !shuffle, order: nextOrder, position: queue.length ? nextOrder.indexOf(current) : -1 })
    },

    cycleRepeat: () => {
        const { repeat } = get()
        set({ repeat: REPEAT_MODES[(REPEAT_MODES.indexOf(repeat) + 1) % REPEAT_MODES.length] })
    },

    // remember the queue and how far into the track it is, for the next page load
    save: (time) => {
        const { queue, order, position, shuffle, repeat, currentTrack, source } = get()
        if (!currentTrack) return
        writeStored(STORAGE_KEY, JSON.stringify({ queue, order, position, shuffle, repeat, source, time: Math.floor(time) }))
    },
}))

export default useAudioStore
