import { MouseSensor, TouchSensor, useSensor, useSensors } from "@dnd-kit/core";

/**
 * Drag sensors that leave scrolling to the finger: a mouse drags after moving
 * 8px, a touch only after holding still for 250ms, so swiping over cards or
 * the calendar scrolls instead of moving them.
 */
export function useDragSensors() {
	return useSensors(
		useSensor(MouseSensor, { activationConstraint: { distance: 8 } }),
		useSensor(TouchSensor, {
			activationConstraint: { delay: 250, tolerance: 8 },
		}),
	);
}
