import "@testing-library/jest-dom";

// Node >= 25 ships its own localStorage, undefined unless --localstorage-file
// is given, and it shadows jsdom's. Give the persisted stores a working one.
if (typeof globalThis.localStorage?.setItem !== "function") {
	const data = new Map<string, string>();
	Object.defineProperty(globalThis, "localStorage", {
		configurable: true,
		value: {
			getItem: (k: string) => data.get(k) ?? null,
			setItem: (k: string, v: string) => data.set(k, String(v)),
			removeItem: (k: string) => data.delete(k),
			clear: () => data.clear(),
			key: (i: number) => [...data.keys()][i] ?? null,
			get length() {
				return data.size;
			},
		} satisfies Storage,
	});
}
