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
import { getRevenueReport, type ReportFilters } from "@/api/reports-service";
import { Money } from "@/components/custom/formatted";
import {
	ChartCard,
	chartColor,
	EmptyChart,
	KpiCard,
	ReportBody,
} from "./report-parts";
import { useReport } from "./use-report";

/** Invoiced vs collected over time, by payment method and by doctor. */
export function RevenueTab({ filters }: { filters: ReportFilters }) {
	const { textGet, formatDate, formatMoney } = useText();
	const { data, loading, error } = useReport(getRevenueReport, filters);

	const tick = (value: string) =>
		formatDate(value.length === 7 ? `${value}-01` : value, "day-month");

	return (
		<ReportBody loading={loading} error={error}>
			{data && (
				<div className="grid gap-4">
					<div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
						<KpiCard label={textGet("reports.revenue.invoiced")}>
							<Money amount={data.totals.invoiced} />
						</KpiCard>
						<KpiCard label={textGet("reports.revenue.collected")}>
							<Money amount={data.totals.collected} />
						</KpiCard>
						<KpiCard label={textGet("reports.revenue.receivable")}>
							<Money amount={data.totals.receivable} />
						</KpiCard>
					</div>

					<ChartCard title={textGet("reports.revenue.series")}>
						{data.series.length === 0 ? (
							<EmptyChart />
						) : (
							<ChartContainer
								className="aspect-auto h-72 w-full"
								config={{
									invoiced: {
										label: textGet("reports.revenue.invoiced"),
										color: "var(--chart-3)",
									},
									collected: {
										label: textGet("reports.revenue.collected"),
										color: "var(--chart-1)",
									},
								}}
							>
								<BarChart data={data.series}>
									<CartesianGrid vertical={false} />
									<XAxis
										dataKey="period"
										tickFormatter={tick}
										tickLine={false}
									/>
									<YAxis
										width={64}
										tickLine={false}
										axisLine={false}
										tickFormatter={(v: number) => formatMoney(v)}
									/>
									<ChartTooltip
										content={
											<ChartTooltipContent
												labelFormatter={(_, p) =>
													tick(String(p?.[0]?.payload?.period ?? ""))
												}
												formatter={(value, name) => (
													<span className="flex w-full justify-between gap-4">
														<span className="text-muted-foreground">
															{textGet(`reports.revenue.${String(name)}`)}
														</span>
														<span className="font-mono font-medium tabular-nums">
															{formatMoney(Number(value))}
														</span>
													</span>
												)}
											/>
										}
									/>
									<Bar
										dataKey="invoiced"
										fill="var(--color-invoiced)"
										radius={3}
									/>
									<Bar
										dataKey="collected"
										fill="var(--color-collected)"
										radius={3}
									/>
								</BarChart>
							</ChartContainer>
						)}
					</ChartCard>

					<div className="grid gap-4 lg:grid-cols-2">
						<ChartCard title={textGet("reports.revenue.by_method")}>
							{data.by_method.length === 0 ? (
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
													<ChartTooltipContent
														nameKey="method"
														hideLabel
														formatter={(value, name) => (
															<span className="flex w-full justify-between gap-4">
																<span className="text-muted-foreground">
																	{textGet(`cash.method.${String(name)}`)}
																</span>
																<span className="font-mono font-medium tabular-nums">
																	{formatMoney(Number(value))}
																</span>
															</span>
														)}
													/>
												}
											/>
											<Pie
												data={data.by_method}
												dataKey="total"
												nameKey="method"
												innerRadius={50}
												stroke="var(--card)"
											>
												{data.by_method.map((m, i) => (
													<Cell key={m.method} fill={chartColor(i)} />
												))}
											</Pie>
										</PieChart>
									</ChartContainer>
									<ul className="grid gap-1 text-sm">
										{data.by_method.map((m, i) => (
											<li key={m.method} className="flex items-center gap-2">
												<span
													aria-hidden
													className="h-2.5 w-2.5 rounded-sm"
													style={{ background: chartColor(i) }}
												/>
												<span className="flex-1">
													{textGet(`cash.method.${m.method}`)}
												</span>
												<Money amount={m.total} className="tabular-nums" />
											</li>
										))}
									</ul>
								</div>
							)}
						</ChartCard>

						<ChartCard title={textGet("reports.revenue.by_doctor")}>
							{data.by_doctor.length === 0 ? (
								<EmptyChart />
							) : (
								<Table>
									<TableHeader>
										<TableRow>
											<TableHead>{textGet("reports.col.doctor")}</TableHead>
											<TableHead className="text-right">
												{textGet("reports.revenue.invoiced")}
											</TableHead>
											<TableHead className="text-right">
												{textGet("reports.revenue.collected")}
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
													<Money amount={d.invoiced} />
												</TableCell>
												<TableCell className="text-right tabular-nums">
													<Money amount={d.collected} />
												</TableCell>
											</TableRow>
										))}
									</TableBody>
								</Table>
							)}
						</ChartCard>
					</div>
				</div>
			)}
		</ReportBody>
	);
}
