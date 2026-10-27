import "@testing-library/jest-dom";

// jsdom logs "not implemented" for scrollTo; pages call it on a new step.
window.scrollTo = () => {};

// jsdom has no matchMedia; the Toaster (next-themes) needs it.
if (!window.matchMedia) {
	window.matchMedia = (query: string) =>
		({
			matches: false,
			media: query,
			onchange: null,
			addListener: () => {},
			removeListener: () => {},
			addEventListener: () => {},
			removeEventListener: () => {},
			dispatchEvent: () => false,
		}) as MediaQueryList;
}
