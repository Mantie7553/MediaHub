// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it } from "vitest"
import { backdropDialog, installModalGestures, isLocked, pullCloses } from "../src/utils/modalGestures"

function dialog(html = "") {
    document.body.innerHTML = `<a href="/somewhere"><dialog class="modal modal-bottom" open>
        <div class="modal-box"><button class="inside">ok</button><input class="field"/></div>${html}
    </dialog></a>`
    const d = document.querySelector("dialog")
    // jsdom may not close dialogs itself
    d.close = () => d.removeAttribute("open")
    return d
}

function press(target, detail = 1) {
    target.dispatchEvent(new Event("pointerdown", { bubbles: true }))
    const click = new MouseEvent("click", { bubbles: true, cancelable: true, detail })
    target.dispatchEvent(click)
    return click
}

describe("closing modals", () => {
    let uninstall
    beforeEach(() => { uninstall = installModalGestures(document, window) })
    afterEach(() => uninstall())

    it("closes on a click outside the box, without following a link around it", () => {
        const d = dialog()
        const click = press(d)
        expect(d.open).toBe(false)
        expect(click.defaultPrevented).toBe(true)
    })

    it("closes on the backdrop's button", () => {
        const d = dialog(`<form method="dialog" class="modal-backdrop"><button class="bd">close</button></form>`)
        press(d.querySelector(".bd"))
        expect(d.open).toBe(false)
    })

    it("stays open for clicks inside the box", () => {
        const d = dialog()
        press(d.querySelector(".inside"))
        expect(d.open).toBe(true)
    })

    it("stays open when a press inside the box lets go outside it", () => {
        const d = dialog()
        d.querySelector(".field").dispatchEvent(new Event("pointerdown", { bubbles: true }))
        d.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, detail: 1 }))
        expect(d.open).toBe(true)
    })

    it("stays open while saving", () => {
        const d = dialog(`<form method="dialog" class="modal-backdrop"><button disabled>close</button></form>`)
        expect(isLocked(d)).toBe(true)
        press(d)
        expect(d.open).toBe(true)
        const locked = dialog()
        locked.dataset.locked = "true"
        press(locked)
        expect(locked.open).toBe(true)
    })

    it("only counts open modals", () => {
        const d = dialog()
        d.removeAttribute("open")
        expect(backdropDialog(d)).toBeNull()
        expect(backdropDialog(d.querySelector(".inside"))).toBeNull()
    })
})

describe("pulling a sheet down", () => {
    it("closes past a distance, or on a quick flick", () => {
        expect(pullCloses(130, 1000)).toBe(true)
        expect(pullCloses(60, 50)).toBe(true)
        expect(pullCloses(60, 400)).toBe(false)
        expect(pullCloses(20, 10)).toBe(false)
    })
})
