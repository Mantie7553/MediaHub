import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import useToastStore, { toast } from "../src/stores/useToastStore"

describe("toasts", () => {
    beforeEach(() => {
        vi.useFakeTimers()
        useToastStore.setState({ toasts: [] })
    })

    afterEach(() => {
        vi.useRealTimers()
    })

    it("shows success and error toasts, errors run through errorMessage", () => {
        toast.success("Saved")
        toast.error({ request: {} }, "Could not save")

        const toasts = useToastStore.getState().toasts
        expect(toasts.map(t => t.type)).toEqual(["success", "error"])
        expect(toasts[0].message).toBe("Saved")
        expect(toasts[1].message).toMatch(/can't reach the server/i)
    })

    it("uses the fallback when the error has nothing better", () => {
        toast.error({ request: {}, response: { status: 500, data: "" } }, "Could not save")
        expect(useToastStore.getState().toasts[0].message).toBe("Could not save")
    })

    it("dismisses after 5 seconds", () => {
        toast.success("Saved")
        vi.advanceTimersByTime(4999)
        expect(useToastStore.getState().toasts).toHaveLength(1)
        vi.advanceTimersByTime(1)
        expect(useToastStore.getState().toasts).toHaveLength(0)
    })

    it("can be dismissed by hand, leaving the others", () => {
        const first = toast.success("One")
        toast.success("Two")
        useToastStore.getState().dismiss(first)
        expect(useToastStore.getState().toasts.map(t => t.message)).toEqual(["Two"])
    })
})
