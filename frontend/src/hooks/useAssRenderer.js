import { useEffect, useRef } from "react"

/**
 * Render ASS subtitles, with their styling and positions, onto a canvas laid over the video.
 * The renderer is only downloaded once a video needs it. The canvas must be a new element
 * for each content, as a canvas can only be handed to one renderer
 * @param {object} videoRef ref to the <video>
 * @param {object} canvasRef ref to the <canvas> over it
 * @param {string|null} content the ASS file, null to render nothing
 * @param {string[]} fonts URLs of the fonts attached to the video
 * @param {function} onError called when the renderer can't start, to fall back to WebVTT
 */
export default function useAssRenderer(videoRef, canvasRef, content, fonts, onError) {
    const onErrorRef = useRef(onError)
    useEffect(() => {
        onErrorRef.current = onError
    })

    useEffect(() => {
        const video = videoRef.current
        const canvas = canvasRef.current
        if (!content || !video || !canvas) return

        // the handler from the render that asked for this content
        const onFail = onErrorRef.current
        let instance = null
        let cancelled = false
        import("jassub")
            .then(({ default: JASSUB }) => {
                if (cancelled) return
                instance = new JASSUB({ video, canvas, subContent: content, fonts })
                return instance.ready
            })
            .catch(err => {
                if (!cancelled) onFail?.(err)
            })

        return () => {
            cancelled = true
            instance?.destroy().catch(() => {})
        }
    }, [videoRef, canvasRef, content, fonts])
}
