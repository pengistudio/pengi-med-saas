import type { ServiceResponse } from "@pengi/shared";
import { useEffect, useState } from "react";
import type { ReportFilters } from "@/api/reports-service";

interface ReportState<T> {
	data: T | null;
	loading: boolean;
	error: boolean;
}

/** Loads a report whenever the filters change; stale responses are dropped. */
export function useReport<T>(
	fetcher: (f: ReportFilters) => Promise<ServiceResponse<T>>,
	filters: ReportFilters,
): ReportState<T> {
	const [state, setState] = useState<ReportState<T>>({
		data: null,
		loading: true,
		error: false,
	});
	const { from, to, doctor_id } = filters;

	// Spinner as soon as the filters change (adjust state during render).
	const key = `${from}|${to}|${doctor_id ?? ""}`;
	const [prevKey, setPrevKey] = useState(key);
	if (prevKey !== key) {
		setPrevKey(key);
		setState((s) => ({ ...s, loading: true, error: false }));
	}

	useEffect(() => {
		let cancelled = false;
		fetcher({ from, to, doctor_id }).then((res) => {
			if (cancelled) return;
			if (res.success && res.data) {
				setState({ data: res.data, loading: false, error: false });
			} else {
				setState({ data: null, loading: false, error: true });
			}
		});
		return () => {
			cancelled = true;
		};
	}, [fetcher, from, to, doctor_id]);

	return state;
}
