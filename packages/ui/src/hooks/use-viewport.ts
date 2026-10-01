import { useSyncExternalStore } from "react";

// Phones are below Tailwind's `md` breakpoint (768px), so this agrees with
// every `md:` / `max-md:` class in the apps.
const PHONE_QUERY = "(max-width: 767px)";

// jsdom (tests) has no matchMedia: without it we assume a desktop.
const phoneQuery = (): MediaQueryList | null =>
	typeof window !== "undefined" && typeof window.matchMedia === "function"
		? window.matchMedia(PHONE_QUERY)
		: null;

/** Whether the viewport is a phone right now, for code outside React. */
export const isPhoneViewport = (): boolean => phoneQuery()?.matches ?? false;

const subscribe = (onChange: () => void) => {
	const query = phoneQuery();
	query?.addEventListener("change", onChange);
	return () => query?.removeEventListener("change", onChange);
};

/**
 * Whether the viewport is a phone or a desktop. Correct on the first render
 * and re-renders when it crosses the breakpoint (resize, rotation).
 */
export function useViewport() {
	const isPhone = useSyncExternalStore(subscribe, isPhoneViewport, () => false);
	return { isPhone, isDesktop: !isPhone };
}
