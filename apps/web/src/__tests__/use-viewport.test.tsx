import { isPhoneViewport, useViewport } from "@pengi/ui";
import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

// A matchMedia whose `matches` the test can flip, notifying listeners.
function stubMatchMedia(initial: boolean) {
	let matches = initial;
	const listeners = new Set<() => void>();
	vi.stubGlobal("matchMedia", (media: string) => ({
		get matches() {
			return matches;
		},
		media,
		addEventListener: (_: string, cb: () => void) => listeners.add(cb),
		removeEventListener: (_: string, cb: () => void) => listeners.delete(cb),
	}));
	return (next: boolean) => {
		matches = next;
		for (const cb of listeners) cb();
	};
}

describe("useViewport", () => {
	afterEach(() => vi.unstubAllGlobals());

	it("is a phone on the first render when the viewport is narrow", () => {
		stubMatchMedia(true);
		const { result } = renderHook(() => useViewport());
		expect(result.current).toEqual({ isPhone: true, isDesktop: false });
	});

	it("re-renders when the viewport crosses the breakpoint", () => {
		const resize = stubMatchMedia(false);
		const { result } = renderHook(() => useViewport());
		expect(result.current.isPhone).toBe(false);

		act(() => resize(true));
		expect(result.current.isPhone).toBe(true);
	});

	it("assumes a desktop without matchMedia", () => {
		vi.stubGlobal("matchMedia", undefined);
		expect(isPhoneViewport()).toBe(false);
	});
});
