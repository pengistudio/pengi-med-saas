import { useText } from "@pengi/shared";
import { Card, CardContent, CardHeader, CardTitle } from "@pengi/ui";
import { ChevronRight } from "lucide-react";
import type React from "react";
import { useNavigate } from "react-router";
import type { DashboardStats } from "@/api/clinical-service";
import { cn } from "@/lib/utils";

const todayISO = () => {
	const d = new Date();
	return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
};

function Delta({ value, labelKey }: { value: number; labelKey: string }) {
	const { textGet } = useText();
	if (value === 0) return null;
	return (
		<span
			className={cn(
				"text-xs font-medium",
				value > 0 ? "text-emerald-600" : "text-muted-foreground",
			)}
		>
			{value > 0 ? "+" : "−"}
			{Math.abs(value)} {textGet(labelKey)}
		</span>
	);
}

function MetricRow({
	label,
	value,
	delta,
	onClick,
}: {
	label: string;
	value: number;
	delta?: React.ReactNode;
	onClick?: () => void;
}) {
	const content = (
		<>
			<span className="text-sm text-muted-foreground">{label}</span>
			<span className="flex items-baseline gap-2">
				{delta}
				<span className="text-lg font-semibold tabular-nums">{value}</span>
				{onClick && (
					<ChevronRight className="h-4 w-4 self-center text-muted-foreground" />
				)}
			</span>
		</>
	);
	const rowClass =
		"flex w-full items-center justify-between gap-3 px-1 py-2.5 text-left";
	return onClick ? (
		<button
			type="button"
			onClick={onClick}
			className={cn(
				rowClass,
				"rounded-md hover:bg-muted/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
			)}
		>
			{content}
		</button>
	) : (
		<div className={rowClass}>{content}</div>
	);
}

/**
 * This week at a glance: one cell per day (click to open that day in the
 * calendar) and the running numbers of the practice.
 */
export function WeekOverview({
	stats,
	className,
}: {
	stats: DashboardStats;
	className?: string;
}) {
	const { textGet } = useText();
	const navigate = useNavigate();
	const today = todayISO();
	const weekTotal = stats.weekly_appointments.reduce((n, d) => n + d.count, 0);

	return (
		<Card className={className}>
			<CardHeader className="pb-2">
				<CardTitle className="text-base">
					{textGet("dashboard.week.title")}
				</CardTitle>
				<p className="text-sm text-muted-foreground">
					{textGet("dashboard.week.total", { count: weekTotal })}
				</p>
			</CardHeader>
			<CardContent className="space-y-4">
				<div className="grid grid-cols-7 gap-1.5">
					{stats.weekly_appointments.map((day) => {
						const isToday = day.date === today;
						return (
							<button
								key={day.date}
								type="button"
								onClick={() =>
									navigate(`/clinical/appointments?date=${day.date}`)
								}
								aria-label={`${day.day}: ${day.count}`}
								className={cn(
									"flex flex-col items-center gap-1 rounded-lg border py-2 transition-colors hover:bg-muted",
									"focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
									isToday
										? "border-primary bg-primary/5"
										: "border-transparent",
								)}
							>
								<span
									className={cn(
										"text-xs",
										isToday
											? "font-semibold text-primary"
											: "text-muted-foreground",
									)}
								>
									{day.day}
								</span>
								<span
									className={cn(
										"text-base font-semibold tabular-nums",
										day.count === 0 && "text-muted-foreground/50",
									)}
								>
									{day.count}
								</span>
							</button>
						);
					})}
				</div>

				<div className="divide-y border-t">
					<MetricRow
						label={textGet("dashboard.stat.today_appointments")}
						value={stats.today_appointments}
						delta={
							<Delta
								value={stats.today_appointments - stats.yesterday_appointments}
								labelKey="dashboard.stat.delta.vs_yesterday"
							/>
						}
					/>
					<MetricRow
						label={textGet("dashboard.stat.monthly_completed")}
						value={stats.monthly_completed}
						delta={
							<Delta
								value={stats.monthly_completed - stats.prev_month_completed}
								labelKey="dashboard.stat.delta.vs_last_month"
							/>
						}
					/>
					<MetricRow
						label={textGet("dashboard.stat.total_patients")}
						value={stats.total_patients}
						delta={
							<Delta
								value={stats.new_patients_this_month}
								labelKey="dashboard.stat.delta.new_this_month"
							/>
						}
						onClick={() => navigate("/clinical")}
					/>
				</div>
			</CardContent>
		</Card>
	);
}
