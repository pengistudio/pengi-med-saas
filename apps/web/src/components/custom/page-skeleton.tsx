import { Skeleton } from "@pengi/ui";

// Invisible for the first 200ms so fast route loads never flash it.
export function PageSkeleton() {
	return (
		<div
			aria-busy="true"
			className="animate-in fade-in-0 duration-(--motion-base) [animation-delay:200ms] [animation-fill-mode:backwards] space-y-4"
		>
			<Skeleton className="h-7 w-48" />
			<Skeleton className="h-4 w-72" />
			<Skeleton className="h-64 w-full rounded-xl" />
		</div>
	);
}
