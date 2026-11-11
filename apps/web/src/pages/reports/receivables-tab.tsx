import { useText } from "@pengi/shared";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@pengi/ui";
import {
	getReceivablesReport,
	type ReportFilters,
} from "@/api/reports-service";
import { FormattedDate, Money } from "@/components/custom/formatted";
import { ChartCard, EmptyChart, KpiCard, ReportBody } from "./report-parts";
import { useReport } from "./use-report";

/** Unpaid and partially paid invoices with their age. */
export function ReceivablesTab({ filters }: { filters: ReportFilters }) {
	const { textGet } = useText();
	const { data, loading, error } = useReport(getReceivablesReport, filters);

	return (
		<ReportBody loading={loading} error={error}>
			{data && (
				<div className="grid gap-4">
					<div className="grid grid-cols-1 gap-4 sm:grid-cols-4">
						<KpiCard label={textGet("reports.receivables.total")}>
							<Money amount={data.totals.balance} />
						</KpiCard>
						{data.buckets.map((b) => (
							<KpiCard
								key={b.key}
								label={textGet(`reports.receivables.bucket.${b.key}`)}
							>
								<Money amount={b.balance} />
							</KpiCard>
						))}
					</div>

					<ChartCard title={textGet("reports.receivables.invoices")}>
						{data.items.length === 0 ? (
							<EmptyChart />
						) : (
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>{textGet("reports.col.invoice")}</TableHead>
										<TableHead>{textGet("reports.col.patient")}</TableHead>
										<TableHead>{textGet("reports.col.date")}</TableHead>
										<TableHead className="text-right">
											{textGet("reports.col.total")}
										</TableHead>
										<TableHead className="text-right">
											{textGet("reports.col.paid")}
										</TableHead>
										<TableHead className="text-right">
											{textGet("reports.col.balance")}
										</TableHead>
										<TableHead className="text-right">
											{textGet("reports.col.age_days")}
										</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{data.items.map((r) => (
										<TableRow key={r.invoice_id}>
											<TableCell>{r.invoice_number}</TableCell>
											<TableCell>{r.patient_name}</TableCell>
											<TableCell>
												<FormattedDate value={r.issue_date} style="medium" />
											</TableCell>
											<TableCell className="text-right tabular-nums">
												<Money amount={r.total} />
											</TableCell>
											<TableCell className="text-right tabular-nums">
												<Money amount={r.paid} />
											</TableCell>
											<TableCell className="text-right font-medium tabular-nums">
												<Money amount={r.balance} />
											</TableCell>
											<TableCell className="text-right tabular-nums">
												{r.age_days}
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
