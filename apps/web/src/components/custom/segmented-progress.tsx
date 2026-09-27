import { cn } from "@/lib/utils";

export interface ProgressSegment {
	key: string;
	count: number;
	/** Background color class of the segment, e.g. `bg-primary`. */
	className: string;
}

/**
 * One bar split into proportional segments, in the order given, followed by a
 * short summary. Used by the boards (tasks, waiting room) to show their state.
 */
export function SegmentedProgress({
	segments,
	summary,
}: {
	segments: ProgressSegment[];
	summary: string;
}) {
	return (
		<div className="flex items-center gap-4">
			<div
				className="flex h-2 flex-1 gap-0.5 overflow-hidden rounded-full bg-muted"
				role="img"
				aria-label={summary}
			>
				{segments
					.filter((s) => s.count > 0)
					.map((s) => (
						<div
							key={s.key}
							className={cn(
								"h-full motion-safe:transition-[flex-grow] motion-safe:duration-500",
								s.className,
							)}
							style={{ flexGrow: s.count }}
						/>
					))}
			</div>
			<p className="shrink-0 text-sm text-muted-foreground tabular-nums">
				{summary}
			</p>
		</div>
	);
}
