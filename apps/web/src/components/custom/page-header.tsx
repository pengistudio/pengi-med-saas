import type React from "react";
import { cn } from "@/lib/utils";

interface PageHeaderProps {
	title: React.ReactNode;
	description?: React.ReactNode;
	/** Page-level actions (e.g. the create button), aligned right on wide screens. */
	actions?: React.ReactNode;
	className?: string;
}

/** The title block at the top of a dashboard page, so every page looks alike. */
export function PageHeader({
	title,
	description,
	actions,
	className,
}: PageHeaderProps) {
	return (
		<div
			className={cn(
				"flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between",
				className,
			)}
		>
			<div>
				<h1 className="text-2xl font-bold tracking-tight">{title}</h1>
				{description && (
					<p className="mt-1 text-sm text-muted-foreground">{description}</p>
				)}
			</div>
			{actions && (
				<div className="flex shrink-0 flex-wrap gap-2 self-start sm:self-auto">
					{actions}
				</div>
			)}
		</div>
	);
}
