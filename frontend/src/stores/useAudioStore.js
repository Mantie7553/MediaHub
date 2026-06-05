import { create } from "zustand"

const useAudioStore = create((set, get) => ({
    currentTrack: null,
    isPlaying: false,
    queue: [],
    currentIndex: -1,

    // play a single track (no queue)
    play: (track) => set({ currentTrack: track, isPlaying: true, queue: [], currentIndex: -1 }),

    // play from a list of tracks starting at an index
    playAlbum: (tracks, index) => set({
        queue: tracks,
        currentIndex: index,
        currentTrack: tracks[index],
        isPlaying: true,
    }),

    pause: () => set({ isPlaying: false }),
    resume: () => set({ isPlaying: true }),

    stop: () => set({ currentTrack: null, isPlaying: false, queue: [], currentIndex: -1 }),

    next: () => {
        const { queue, currentIndex } = get()
        if (queue.length === 0 || currentIndex >= queue.length - 1) {
            set({ currentTrack: null, isPlaying: false, queue: [], currentIndex: -1 })
            return
        }
        const nextIndex = currentIndex + 1
        set({ currentIndex: nextIndex, currentTrack: queue[nextIndex], isPlaying: true })
    },

    prev: () => {
        const { queue, currentIndex } = get()
        if (queue.length === 0 || currentIndex <= 0) return
        const prevIndex = currentIndex - 1
        set({ currentIndex: prevIndex, currentTrack: queue[prevIndex], isPlaying: true })
    },
}))

export default useAudioStore