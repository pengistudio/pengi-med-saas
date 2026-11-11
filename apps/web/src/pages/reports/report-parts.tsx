import { useText } from "@pengi/shared";
import { Card, CardContent, CardHeader, CardTitle, Skeleton } from "@pengi/ui";
import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

/** Series colors: the theme's chart tokens (they follow light/dark). */
export const CHART_COLORS = [
	"var(--chart-1)",
	"var(--chart-3)",
	"var(--chart-5)",
	"var(--chart-2)",
	"var(--chart-4)",
] as const;

export const chartColor = (i: number) => CHART_COLORS[i % CHART_COLORS.length];

export function KpiCard({
	label,
	children,
	className,
}: {
	label: string;
	children: ReactNode;
	className?: string;
}) {
	return (
		<Card size="sm">
			<CardHeader>
				<CardTitle className="text-sm font-medium text-muted-foreground">
					{label}
				</CardTitle>
			</CardHeader>
			<CardContent>
				<p className={cn("text-2xl font-semibold tabular-nums", className)}>
					{children}
				</p>
			</CardContent>
		</Card>
	);
}

export function ChartCard({
	title,
	children,
	className,
}: {
	title: string;
	children: ReactNode;
	className?: string;
}) {
	return (
		<Card className={className}>
			<CardHeader>
				<CardTitle className="text-base">{title}</CardTitle>
			</CardHeader>
			<CardContent>{children}</CardContent>
		</Card>
	);
}

export function EmptyChart() {
	const { textGet } = useText();
	return (
		<p className="py-10 text-center text-sm text-muted-foreground">
			{textGet("reports.empty")}
		</p>
	);
}

/** Loading, error and "nothing to show" states around a report's content. */
export function ReportBody({
	loading,
	error,
	children,
}: {
	loading: boolean;
	error: boolean;
	children: ReactNode;
}) {
	const { textGet } = useText();
	if (loading) {
		return (
			<div className="grid gap-4" aria-busy="true">
				<div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
					<Skeleton className="h-24" />
					<Skeleton className="h-24" />
					<Skeleton className="h-24" />
				</div>
				<Skeleton className="h-72" />
			</div>
		);
	}
	if (error) {
		return (
			<p role="alert" className="py-10 text-center text-sm text-destructive">
				{textGet("reports.error")}
			</p>
		);
	}
	return <>{children}</>;
}
