// Where a block was released when dropped on another day, by appointment ID.
// The block remounts in the new column and slides in from this point.
const landings = new Map<number, { left: number; top: number }>();

export function rememberLanding(
	id: number,
	rect: { left: number; top: number },
) {
	if (window.matchMedia?.("(prefers-reduced-motion: reduce)").matches) return;
	landings.set(id, { left: rect.left, top: rect.top });
}

export function peekLanding(id: number) {
	return landings.get(id);
}

export function clearLanding(id: number) {
	landings.delete(id);
}
