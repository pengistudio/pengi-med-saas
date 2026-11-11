import { useText } from "@pengi/shared";
import {
	ChartContainer,
	ChartTooltip,
	ChartTooltipContent,
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@pengi/ui";
import {
	Bar,
	BarChart,
	CartesianGrid,
	Cell,
	Pie,
	PieChart,
	XAxis,
	YAxis,
} from "recharts";
import {
	getAppointmentsReport,
	type ReportFilters,
} from "@/api/reports-service";
import {
	ChartCard,
	chartColor,
	EmptyChart,
	KpiCard,
	ReportBody,
} from "./report-parts";
import { useReport } from "./use-report";

/** Appointments by status, doctor and type, plus patients new vs returning. */
export function AppointmentsTab({ filters }: { filters: ReportFilters }) {
	const { textGet, formatDate } = useText();
	const { data, loading, error } = useReport(getAppointmentsReport, filters);

	const tick = (value: string) =>
		formatDate(value.length === 7 ? `${value}-01` : value, "day-month");

	return (
		<ReportBody loading={loading} error={error}>
			{data && (
				<div className="grid gap-4">
					<div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
						<KpiCard label={textGet("reports.appointments.total")}>
							{data.totals.total}
						</KpiCard>
						<KpiCard label={textGet("reports.appointments.no_show_rate")}>
							{`${(data.totals.no_show_rate * 100).toFixed(1)}%`}
						</KpiCard>
						<KpiCard label={textGet("reports.appointments.new_patients")}>
							{data.patients.new}
						</KpiCard>
						<KpiCard label={textGet("reports.appointments.returning_patients")}>
							{data.patients.recurring}
						</KpiCard>
					</div>

					<ChartCard title={textGet("reports.appointments.series")}>
						{data.series.length === 0 ? (
							<EmptyChart />
						) : (
							<ChartContainer
								className="aspect-auto h-64 w-full"
								config={{
									total: {
										label: textGet("reports.appointments.total"),
										color: "var(--chart-1)",
									},
								}}
							>
								<BarChart data={data.series}>
									<CartesianGrid vertical={false} />
									<XAxis dataKey="day" tickFormatter={tick} tickLine={false} />
									<YAxis
										width={32}
										allowDecimals={false}
										tickLine={false}
										axisLine={false}
									/>
									<ChartTooltip content={<ChartTooltipContent />} />
									<Bar dataKey="total" fill="var(--color-total)" radius={3} />
								</BarChart>
							</ChartContainer>
						)}
					</ChartCard>

					<div className="grid gap-4 lg:grid-cols-2">
						<ChartCard title={textGet("reports.appointments.by_status")}>
							{data.by_status.length === 0 ? (
								<EmptyChart />
							) : (
								<div className="grid gap-3">
									<ChartContainer
										className="aspect-auto h-56 w-full"
										config={{}}
									>
										<PieChart>
											<ChartTooltip
												content={
													<ChartTooltipContent hideLabel nameKey="status" />
												}
											/>
											<Pie
												data={data.by_status}
												dataKey="count"
												nameKey="status"
												innerRadius={50}
												stroke="var(--card)"
											>
												{data.by_status.map((s, i) => (
													<Cell key={s.status} fill={chartColor(i)} />
												))}
											</Pie>
										</PieChart>
									</ChartContainer>
									<ul className="grid gap-1 text-sm">
										{data.by_status.map((s, i) => (
											<li key={s.status} className="flex items-center gap-2">
												<span
													aria-hidden
													className="h-2.5 w-2.5 rounded-sm"
													style={{ background: chartColor(i) }}
												/>
												<span className="flex-1">
													{textGet(`reports.appointments.status.${s.status}`)}
												</span>
												<span className="tabular-nums">{s.count}</span>
											</li>
										))}
									</ul>
								</div>
							)}
						</ChartCard>

						<ChartCard title={textGet("reports.appointments.by_type")}>
							{data.by_type.length === 0 ? (
								<EmptyChart />
							) : (
								<ChartContainer
									className="aspect-auto h-64 w-full"
									config={{
										count: {
											label: textGet("reports.appointments.total"),
											color: "var(--chart-3)",
										},
									}}
								>
									<BarChart data={data.by_type} layout="vertical">
										<CartesianGrid horizontal={false} />
										<YAxis
											dataKey="type_name"
											type="category"
											width={110}
											tickLine={false}
											axisLine={false}
										/>
										<XAxis type="number" allowDecimals={false} hide />
										<ChartTooltip content={<ChartTooltipContent hideLabel />} />
										<Bar dataKey="count" fill="var(--color-count)" radius={3} />
									</BarChart>
								</ChartContainer>
							)}
						</ChartCard>
					</div>

					<ChartCard title={textGet("reports.appointments.by_doctor")}>
						{data.by_doctor.length === 0 ? (
							<EmptyChart />
						) : (
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>{textGet("reports.col.doctor")}</TableHead>
										<TableHead className="text-right">
											{textGet("reports.col.appointments")}
										</TableHead>
										<TableHead className="text-right">
											{textGet("reports.col.completed")}
										</TableHead>
										<TableHead className="text-right">
											{textGet("reports.col.no_show")}
										</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{data.by_doctor.map((d) => (
										<TableRow key={d.doctor_id}>
											<TableCell>
												{d.doctor_name || textGet("reports.col.no_doctor")}
											</TableCell>
											<TableCell className="text-right tabular-nums">
												{d.total}
											</TableCell>
											<TableCell className="text-right tabular-nums">
												{d.completed}
											</TableCell>
											<TableCell className="text-right tabular-nums">
												{d.no_show}
											</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
						)}
					</ChartCard>
				</div>
			)}
		</ReportBody>
	);
}
