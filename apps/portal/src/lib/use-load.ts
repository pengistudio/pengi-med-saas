import type { ServiceResponse } from "@pengi/shared";
import React from "react";

export type LoadState<T> =
	| { status: "loading" }
	| { status: "error" }
	| { status: "ready"; data: T };

/**
 * Runs `load` on mount and tracks its result. `reload` runs it again and
 * keeps showing the current data until the new answer arrives.
 */
export function useReloadableLoad<T>(
	load: () => Promise<ServiceResponse<T>>,
): [LoadState<T>, () => void] {
	const [state, setState] = React.useState<LoadState<T>>({ status: "loading" });
	const [version, setVersion] = React.useState(0);
	const loadRef = React.useRef(load);

	React.useEffect(() => {
		let active = true;
		loadRef.current().then((res) => {
			if (!active) return;
			setState(
				res.success ? { status: "ready", data: res.data } : { status: "error" },
			);
		});
		return () => {
			active = false;
		};
	}, [version]);

	const reload = React.useCallback(() => setVersion((v) => v + 1), []);
	return [state, reload];
}

/** Runs `load` once on mount and tracks its result. */
export function useLoad<T>(load: () => Promise<ServiceResponse<T>>) {
	return useReloadableLoad(load)[0];
}
