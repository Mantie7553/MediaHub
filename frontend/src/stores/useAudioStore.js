import { create } from "zustand"

const useAudioStore = create((set) => ({
    currentTrack: null,
    isPlaying: false,

    play: (track) => set({ currentTrack: track, isPlaying: true }),
    pause: () => set({ isPlaying: false }),
    resume: () => set({ isPlaying: true }),
    stop: () => set({ currentTrack: null, isPlaying: false }),
}))

export default useAudioStore