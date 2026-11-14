import type { CDPSession, Page } from "@playwright/test";
type Contact = { id: number; x: number; y: number };

// Chromium exercises trusted browser touch/pen input. WebKit lacks a multi-touch
// automation API, so it receives pointer plus legacy touch events for app logic.
export class TabletInput {
  private touches = new Map<number, Contact>();
  private cdp?: CDPSession;
  constructor(private page: Page) {}
  async native() {
    this.cdp = await this.page.context().newCDPSession(this.page);
  }
  async touch(
    kind: "down" | "move" | "up" | "cancel",
    id: number,
    x: number,
    y: number,
  ) {
    const contact = { id, x, y };
    if (kind === "up" || kind === "cancel") this.touches.delete(id);
    else this.touches.set(id, contact);
    if (kind === "cancel") this.touches.clear();
    const active = [...this.touches.values()];
    if (this.cdp) {
      await this.cdp.send("Input.dispatchTouchEvent", {
        type: (
          {
            down: "touchStart",
            move: "touchMove",
            up: "touchEnd",
            cancel: "touchCancel",
          } as const
        )[kind],
        touchPoints: active.map((point) => ({
          ...point,
          radiusX: 1,
          radiusY: 1,
          force: 1,
        })),
      });
    } else await this.dispatch(kind, contact, "touch", active);
  }
  async pen(
    kind: "down" | "move" | "up" | "cancel",
    x: number,
    y: number,
    pressure = 0.5,
  ) {
    if (this.cdp && kind !== "cancel") {
      await this.cdp.send("Input.dispatchMouseEvent", {
        type: (
          {
            down: "mousePressed",
            move: "mouseMoved",
            up: "mouseReleased",
          } as const
        )[kind],
        x,
        y,
        button: "left",
        buttons: kind === "up" ? 0 : 1,
        clickCount: kind === "move" ? 0 : 1,
        pointerType: "pen",
        force: kind === "up" ? 0 : pressure,
      });
    } else
      await this.dispatch(
        kind,
        { id: 99, x, y },
        "pen",
        kind === "up" || kind === "cancel" ? [] : [{ id: 99, x, y }],
        pressure,
      );
  }
  private async dispatch(
    kind: string,
    contact: Contact,
    pointerType: string,
    active: Contact[],
    pressure = 0.5,
  ) {
    await this.page.locator(".konvajs-content").evaluate(
      (surface, { kind, contact, pointerType, active, pressure }) => {
        surface.dispatchEvent(
          new PointerEvent(`pointer${kind}`, {
            bubbles: true,
            cancelable: true,
            pointerId: contact.id,
            pointerType,
            clientX: contact.x,
            clientY: contact.y,
            button: 0,
            buttons: kind === "up" || kind === "cancel" ? 0 : 1,
            pressure: kind === "up" || kind === "cancel" ? 0 : pressure,
            isPrimary: true,
          }),
        );
        const touch = (point: Contact) => ({
          identifier: point.id,
          clientX: point.x,
          clientY: point.y,
          pageX: point.x,
          pageY: point.y,
          target: surface,
          touchType: pointerType === "pen" ? "stylus" : "direct",
        });
        const event = new Event(
          {
            down: "touchstart",
            move: "touchmove",
            up: "touchend",
            cancel: "touchcancel",
          }[kind]!,
          { bubbles: true, cancelable: true },
        );
        Object.defineProperties(event, {
          touches: { value: active.map(touch) },
          targetTouches: { value: active.map(touch) },
          changedTouches: { value: [touch(contact)] },
        });
        surface.dispatchEvent(event);
      },
      { kind, contact, pointerType, active, pressure },
    );
  }
  async drag(
    from: { x: number; y: number },
    to: { x: number; y: number },
    type: "touch" | "pen" = "touch",
  ) {
    const step = (kind: "down" | "move" | "up", x: number, y: number) =>
      type === "touch" ? this.touch(kind, 1, x, y) : this.pen(kind, x, y);
    await step("down", from.x, from.y);
    for (let index = 1; index <= 8; index++)
      await step(
        "move",
        from.x + ((to.x - from.x) * index) / 8,
        from.y + ((to.y - from.y) * index) / 8,
      );
    await step("up", to.x, to.y);
  }
}
