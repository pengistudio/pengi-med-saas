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
import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";
import { getClinicalReport, type ReportFilters } from "@/api/reports-service";
import { ChartCard, EmptyChart, ReportBody } from "./report-parts";
import { useReport } from "./use-report";

/** Top diagnoses and the clinical output of each doctor. */
export function ClinicalTab({ filters }: { filters: ReportFilters }) {
	const { textGet } = useText();
	const { data, loading, error } = useReport(getClinicalReport, filters);

	return (
		<ReportBody loading={loading} error={error}>
			{data && (
				<div className="grid gap-4">
					<ChartCard title={textGet("reports.clinical.top_diagnoses")}>
						{data.top_diagnoses.length === 0 ? (
							<EmptyChart />
						) : (
							<ChartContainer
								className="aspect-auto w-full"
								style={{
									height: Math.max(200, data.top_diagnoses.length * 28),
								}}
								config={{
									count: {
										label: textGet("reports.col.count"),
										color: "var(--chart-1)",
									},
								}}
							>
								<BarChart data={data.top_diagnoses} layout="vertical">
									<CartesianGrid horizontal={false} />
									<YAxis
										dataKey="code"
										type="category"
										width={64}
										tickLine={false}
										axisLine={false}
									/>
									<XAxis type="number" allowDecimals={false} hide />
									<ChartTooltip
										content={
											<ChartTooltipContent
												labelFormatter={(_, p) => {
													const d = p?.[0]?.payload;
													return d ? `${d.code} ${d.title}` : "";
												}}
											/>
										}
									/>
									<Bar dataKey="count" fill="var(--color-count)" radius={3} />
								</BarChart>
							</ChartContainer>
						)}
					</ChartCard>

					<ChartCard title={textGet("reports.clinical.by_doctor")}>
						{data.by_doctor.length === 0 ? (
							<EmptyChart />
						) : (
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>{textGet("reports.col.doctor")}</TableHead>
										<TableHead className="text-right">
											{textGet("reports.col.consultations")}
										</TableHead>
										<TableHead className="text-right">
											{textGet("reports.col.prescriptions")}
										</TableHead>
										<TableHead className="text-right">
											{textGet("reports.col.exam_orders")}
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
												{d.consultations}
											</TableCell>
											<TableCell className="text-right tabular-nums">
												{d.prescriptions}
											</TableCell>
											<TableCell className="text-right tabular-nums">
												{d.exam_orders}
											</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
						)}
					</ChartCard>

					{data.top_diagnoses.length > 0 && (
						<ChartCard title={textGet("reports.clinical.diagnoses_table")}>
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>{textGet("reports.col.code")}</TableHead>
										<TableHead>{textGet("reports.col.description")}</TableHead>
										<TableHead className="text-right">
											{textGet("reports.col.count")}
										</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{data.top_diagnoses.map((d) => (
										<TableRow key={d.code}>
											<TableCell>{d.code}</TableCell>
											<TableCell>{d.title}</TableCell>
											<TableCell className="text-right tabular-nums">
												{d.count}
											</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
						</ChartCard>
					)}
				</div>
			)}
		</ReportBody>
	);
}
