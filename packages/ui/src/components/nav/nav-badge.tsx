import { cn } from "../../lib/utils";

/** Counter pill for a nav item: nothing at 0, "99+" above 99. */
export function NavBadge({
	count,
	className,
}: {
	count?: number;
	className?: string;
}) {
	if (!count || count < 1) return null;
	return (
		<span
			className={cn(
				"flex h-5 min-w-5 shrink-0 items-center justify-center rounded-full bg-primary px-1 text-[0.6875rem] leading-none font-semibold text-primary-foreground tabular-nums",
				className,
			)}
		>
			{count > 99 ? "99+" : count}
		</span>
	);
}
