import * as React from "react";

/**
 * Calls `task` every `intervalMs` while `enabled`, without overlapping: the
 * next run is scheduled only after the previous one settles. It skips runs
 * while the tab is hidden and stops on unmount. Runs on the first tick, not
 * immediately: load the first time yourself.
 */
export function usePolling(
	task: () => Promise<unknown>,
	intervalMs: number,
	enabled = true,
) {
	const taskRef = React.useRef(task);
	React.useEffect(() => {
		taskRef.current = task;
	}, [task]);

	React.useEffect(() => {
		if (!enabled) return;
		let stopped = false;
		let timer: ReturnType<typeof setTimeout>;
		const schedule = () => {
			timer = setTimeout(tick, intervalMs);
		};
		const tick = async () => {
			if (stopped) return;
			if (!document.hidden) {
				try {
					await taskRef.current();
				} catch {
					// A failed poll waits for the next tick.
				}
			}
			if (!stopped) schedule();
		};
		schedule();
		return () => {
			stopped = true;
			clearTimeout(timer);
		};
	}, [intervalMs, enabled]);
}
