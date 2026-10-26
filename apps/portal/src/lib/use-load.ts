import type { ServiceResponse } from "@pengi/shared";
import React from "react";

export type LoadState<T> =
	| { status: "loading" }
	| { status: "error" }
	| { status: "ready"; data: T };

/** Runs `load` once on mount and tracks its result. */
export function useLoad<T>(load: () => Promise<ServiceResponse<T>>) {
	const [state, setState] = React.useState<LoadState<T>>({ status: "loading" });
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
	}, []);

	return state;
}
