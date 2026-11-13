import type { ReactNode } from "react";
import { cn } from "../lib/utils";
import { Illustration, type IllustrationName } from "./illustration";

export interface EmptyStateProps {
	illustration: IllustrationName;
	title: ReactNode;
	description?: ReactNode;
	/** Button or link rendered under the text. */
	action?: ReactNode;
	/** `sm` for tables, cards and panels; `md` (default) for whole pages. */
	size?: "sm" | "md";
	className?: string;
}

/**
 * Illustration + title + description + optional action, for empty, error and
 * flow screens. Callers pass already-translated text (`textGet`).
 */
export function EmptyState({
	illustration,
	title,
	description,
	action,
	size = "md",
	className,
}: EmptyStateProps) {
	const small = size === "sm";
	return (
		<div
			className={cn(
				"flex flex-col items-center text-center",
				small ? "gap-1 py-6" : "gap-2 py-12",
				className,
			)}
		>
			<Illustration name={illustration} className={small ? "w-32" : "w-48"} />
			<p className={cn("font-semibold", small ? "text-sm" : "text-lg")}>
				{title}
			</p>
			{description ? (
				<p className="max-w-sm text-sm text-muted-foreground">{description}</p>
			) : null}
			{action ? <div className="mt-3">{action}</div> : null}
		</div>
	);
}
