import { useText } from "@pengi/shared";
import { Button, Card, CardContent, CardHeader, CardTitle } from "@pengi/ui";
import React from "react";
import { useNavigate } from "react-router";
import {
	getAllInvoices,
	type Invoice,
	processInvoiceSRI,
} from "@/api/billing-service";
import { InvoiceStatusBadge } from "@/components/custom/billing/invoice-status-badge";

const RECENT_LIMIT = 5;

const invoiceNumber = (i: Invoice) =>
	`${i.establishment_code}-${i.emission_point_code}-${i.sequential}`;

/**
 * The last invoices issued. A rejected or failed one can be resent from its
 * status badge without leaving the dashboard.
 */
export function RecentInvoicesCard({ canRetry }: { canRetry: boolean }) {
	const { textGet, formatDate, formatMoney } = useText();
	const navigate = useNavigate();
	const [invoices, setInvoices] = React.useState<Invoice[]>([]);

	const load = React.useCallback(async () => {
		const res = await getAllInvoices({ limit: RECENT_LIMIT });
		if (res.success && res.data) setInvoices(res.data.items);
	}, []);

	React.useEffect(() => {
		load();
	}, [load]);

	const retry = async (invoice: Invoice) => {
		const res = await processInvoiceSRI(invoice.ID);
		if (res.success) await load();
	};

	return (
		<Card>
			<CardHeader className="flex flex-row items-start justify-between gap-4">
				<CardTitle className="text-base">
					{textGet("dashboard.recent_invoices.title")}
				</CardTitle>
				<Button variant="ghost" size="sm" onClick={() => navigate("/billing")}>
					{textGet("dashboard.recent_invoices.view_all")}
				</Button>
			</CardHeader>
			<CardContent>
				{invoices.length === 0 ? (
					<p className="rounded-lg border border-dashed p-6 text-sm text-muted-foreground">
						{textGet("dashboard.recent_invoices.empty")}
					</p>
				) : (
					<ul className="divide-y">
						{invoices.map((invoice) => {
							const patientName = invoice.patient
								? `${invoice.patient.first_name} ${invoice.patient.last_name}`
								: textGet("billing.invoice.final_consumer");
							const retryable = [
								"failed",
								"rejected",
								"connection_error",
							].includes(invoice.status);
							return (
								<li
									key={invoice.ID}
									className="flex items-center justify-between gap-3 py-2.5"
								>
									<div className="min-w-0 flex-1">
										<p className="truncate text-sm font-medium">
											{patientName}
										</p>
										<p className="text-xs text-muted-foreground tabular-nums">
											{invoiceNumber(invoice)}, {formatDate(invoice.CreatedAt)}
										</p>
									</div>
									<span className="text-sm font-semibold tabular-nums">
										{formatMoney(invoice.total)}
									</span>
									<InvoiceStatusBadge
										status={invoice.status}
										errorMessage={invoice.error_message}
										errorCode={invoice.error_code}
										onRetry={
											canRetry && retryable ? () => retry(invoice) : undefined
										}
									/>
								</li>
							);
						})}
					</ul>
				)}
			</CardContent>
		</Card>
	);
}
