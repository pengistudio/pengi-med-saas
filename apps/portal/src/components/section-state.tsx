import { useText } from "@pengi/shared";
import { Skeleton } from "@pengi/ui";
import type React from "react";
import type { LoadState } from "@/lib/use-load";

/** Loading, error and empty states shared by every section. */
export function SectionState<T>({
	state,
	emptyKey,
	isEmpty,
	children,
}: {
	state: LoadState<T>;
	emptyKey: string;
	isEmpty: (data: T) => boolean;
	children: (data: T) => React.ReactNode;
}) {
	const { textGet } = useText();

	if (state.status === "loading") {
		return (
			<div className="space-y-3">
				<Skeleton className="h-16 w-full" />
				<Skeleton className="h-16 w-full" />
			</div>
		);
	}
	if (state.status === "error") {
		return (
			<p className="text-muted-foreground text-sm">
				{textGet("portal.section.error")}
			</p>
		);
	}
	if (isEmpty(state.data)) {
		return <p className="text-muted-foreground text-sm">{textGet(emptyKey)}</p>;
	}
	return <>{children(state.data)}</>;
}
