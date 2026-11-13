import type Konva from "konva";
import type {
  PointerEvent as ReactPointerEvent,
  TouchEvent as ReactTouchEvent,
  RefObject,
} from "react";
import { useEffect, useRef, useState } from "react";
import type { Point } from "../../model";
import type { Camera } from "./types";
import { touchCamera } from "./touchNavigation";

export type CanvasInteraction = { cancel: () => void; finish: () => void };
type Event = Konva.KonvaEventObject<PointerEvent>;
type Contact = Point & { type: string };
type Navigation = {
  ids: [number, number];
  start: [Point, Point];
  camera: Camera;
};
type Props = {
  stage: RefObject<Konva.Stage | null>;
  scene: RefObject<CanvasInteraction | null>;
  camera: Camera;
  onCamera: (camera: Camera) => void;
  start: (event: Event) => void;
  move: (event: Event) => void;
  end: (event: PointerEvent) => void;
  cancel: () => void;
};

// Pointer events own drawing/navigation. Konva retains its native mouse/touch
// drag and Transformer handlers; we cancel those explicitly before navigating.
export default function useCanvasInput({
  stage,
  scene,
  camera,
  onCamera,
  start,
  move,
  end,
  cancel,
}: Props) {
  const contacts = useRef(new Map<number, Contact>());
  const lastMouse = useRef<PointerEvent | undefined>(undefined);
  const blocked = useRef(new Set<number>());
  const primary = useRef<number | undefined>(undefined);
  const pen = useRef<number | undefined>(undefined);
  const navigation = useRef<Navigation | undefined>(undefined);
  const suspended = useRef(false);
  const [navigating, setNavigating] = useState(false);
  const [coarse, setCoarse] = useState(
    () => matchMedia("(any-pointer: coarse)").matches,
  );
  useEffect(() => {
    const interrupt = () => {
      if (!contacts.current.size) return;
      const mouse = lastMouse.current;
      if (mouse && contacts.current.get(mouse.pointerId)?.type === "mouse") {
        finishPointer(mouse);
        scene.current?.finish();
      } else {
        cancelInteraction();
      }
      for (const id of contacts.current.keys()) blocked.current.add(id);
      contacts.current.clear();
      primary.current = undefined;
      pen.current = undefined;
      navigation.current = undefined;
      suspended.current = false;
      setNavigating(false);
    };
    const hidden = () => {
      if (document.hidden) interrupt();
    };
    const mouseRelease = (event: PointerEvent) => {
      if (event.pointerType === "mouse") finishPointer(event);
    };
    // Fallback when release targets something outside the stage after capture loss.
    window.addEventListener("pointerup", mouseRelease);
    window.addEventListener("blur", interrupt);
    window.addEventListener("resize", interrupt);
    document.addEventListener("visibilitychange", hidden);
    return () => {
      window.removeEventListener("pointerup", mouseRelease);
      window.removeEventListener("blur", interrupt);
      window.removeEventListener("resize", interrupt);
      document.removeEventListener("visibilitychange", hidden);
    };
  });
  function surface(event: ReactPointerEvent<HTMLDivElement>) {
    return (
      event.target instanceof Element &&
      Boolean(event.target.closest(".konvajs-content"))
    );
  }
  function point(event: PointerEvent): Point {
    const box = stage.current!.container().getBoundingClientRect();
    return { x: event.clientX - box.left, y: event.clientY - box.top };
  }
  function cancelInteraction() {
    cancel();
    scene.current?.cancel();
  }
  function finishPointer(input: PointerEvent) {
    if (primary.current !== input.pointerId) return;
    if (allowed(input)) {
      stage.current?.setPointersPositions(input);
      end(input);
    }
    primary.current = undefined;
    pen.current = undefined;
    contacts.current.delete(input.pointerId);
  }
  function down(event: ReactPointerEvent<HTMLDivElement>) {
    if (!surface(event)) return;
    const input = event.nativeEvent;
    if (input.pointerType === "mouse") lastMouse.current = input;
    blocked.current.delete(input.pointerId);
    if (input.pointerType !== "mouse") setCoarse(true);
    if (pen.current !== undefined && input.pointerId !== pen.current) {
      blocked.current.add(input.pointerId);
      event.preventDefault();
      event.stopPropagation();
      return;
    }
    if (input.pointerType === "pen") {
      if (contacts.current.size) {
        cancelInteraction();
        for (const id of contacts.current.keys()) blocked.current.add(id);
        contacts.current.clear();
        navigation.current = undefined;
        suspended.current = false;
        setNavigating(false);
      }
      pen.current = input.pointerId;
    }
    contacts.current.set(input.pointerId, {
      ...point(input),
      type: input.pointerType,
    });
    if (contacts.current.size === 1 && !suspended.current)
      primary.current = input.pointerId;
    const touches = [...contacts.current].filter(
      ([, contact]) => contact.type === "touch",
    );
    if (touches.length >= 2 && !navigation.current) {
      cancelInteraction();
      primary.current = undefined;
      suspended.current = true;
      navigation.current = {
        ids: [touches[0][0], touches[1][0]],
        start: [touches[0][1], touches[1][1]],
        camera,
      };
      setNavigating(true);
    }
    if (input.isTrusted && input.target instanceof Element)
      input.target.setPointerCapture?.(input.pointerId);
  }
  function captureMove(event: ReactPointerEvent<HTMLDivElement>) {
    const input = event.nativeEvent;
    const contact = contacts.current.get(input.pointerId);
    if (contact?.type === "mouse") {
      if (input.buttons === 0) {
        finishPointer(lastMouse.current || input);
        scene.current?.finish();
        return;
      }
      lastMouse.current = input;
    }
    if (contact)
      contacts.current.set(input.pointerId, {
        ...point(input),
        type: contact.type,
      });
    if (blocked.current.has(input.pointerId)) {
      event.stopPropagation();
      return;
    }
    const nav = navigation.current;
    if (!nav) return;
    const first = contacts.current.get(nav.ids[0]),
      second = contacts.current.get(nav.ids[1]);
    if (first && second)
      onCamera(touchCamera(nav.camera, nav.start, [first, second]));
    event.preventDefault();
    event.stopPropagation();
  }
  function captureUp(event: ReactPointerEvent<HTMLDivElement>) {
    const id = event.pointerId;
    if (blocked.current.delete(id)) {
      event.stopPropagation();
      return;
    }
    contacts.current.delete(id);
    if (navigation.current && contacts.current.size < 2) {
      navigation.current = undefined;
      setNavigating(false);
    }
    if (suspended.current) {
      if (!contacts.current.size) suspended.current = false;
      event.stopPropagation();
      return;
    }
    finishPointer(event.nativeEvent);
  }
  function interrupted(event: ReactPointerEvent<HTMLDivElement>) {
    const contact = contacts.current.get(event.pointerId);
    if (!contact) return;
    if (contact.type === "mouse") {
      finishPointer(lastMouse.current || event.nativeEvent);
      scene.current?.finish();
      return;
    }
    cancelInteraction();
    for (const id of contacts.current.keys())
      if (id !== event.pointerId) blocked.current.add(id);
    contacts.current.clear();
    primary.current = undefined;
    pen.current = undefined;
    navigation.current = undefined;
    suspended.current = false;
    setNavigating(false);
  }
  function nativeTouch(event: ReactTouchEvent<HTMLDivElement>) {
    if (
      !(event.target instanceof Element) ||
      !event.target.closest(".konvajs-content")
    )
      return;
    const stylus = Array.from(event.nativeEvent.changedTouches).every(
      (touch) =>
        (touch as Touch & { touchType?: string }).touchType === "stylus",
    );
    if (
      suspended.current ||
      navigation.current ||
      (pen.current !== undefined && !stylus)
    ) {
      event.stopPropagation();
    }
  }
  function allowed(event: PointerEvent) {
    return (
      !suspended.current &&
      !blocked.current.has(event.pointerId) &&
      (pen.current === undefined || pen.current === event.pointerId) &&
      (primary.current === undefined || primary.current === event.pointerId)
    );
  }
  return {
    coarse,
    navigating,
    allowed,
    capture: {
      onPointerDownCapture: down,
      onPointerMoveCapture: captureMove,
      onPointerUpCapture: captureUp,
      onPointerCancelCapture: interrupted,
      onLostPointerCapture: interrupted,
      onTouchStartCapture: nativeTouch,
      onTouchMoveCapture: nativeTouch,
      onTouchEndCapture: nativeTouch,
      onTouchCancelCapture: nativeTouch,
    },
    down(event: Event) {
      if (allowed(event.evt) && primary.current === event.evt.pointerId)
        start(event);
    },
    move(event: Event) {
      if (!allowed(event.evt)) return;
      if (
        primary.current !== event.evt.pointerId &&
        event.evt.pointerType === "touch"
      )
        return;
      if (
        event.evt.pointerType === "pen" &&
        primary.current === event.evt.pointerId
      ) {
        for (const sample of event.evt.getCoalescedEvents?.() || []) {
          stage.current?.setPointersPositions(sample);
          move(event);
        }
        stage.current?.setPointersPositions(event.evt);
      }
      move(event);
    },
  };
}
