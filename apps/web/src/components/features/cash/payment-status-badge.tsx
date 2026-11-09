import { useText } from "@pengi/shared";
import { Badge } from "@pengi/ui";
import type { Invoice } from "@/api/billing-service";
import { paymentStatusLabel } from "./cash-labels";

const STYLES = {
	unpaid: "bg-destructive/10 text-destructive",
	partial: "bg-amber-500/15 text-amber-700 dark:text-amber-400",
	paid: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-400",
} as const;

/**
 * Collection status of an invoice (independent of its SRI status). Null for
 * invoices from before the first cash session: shows "—".
 */
export function PaymentStatusBadge({
	status,
}: {
	status: Invoice["payment_status"];
}) {
	const { textGet } = useText();
	if (!status) return <span className="text-muted-foreground">—</span>;
	return (
		<Badge variant="secondary" className={STYLES[status]}>
			{paymentStatusLabel(textGet, status)}
		</Badge>
	);
}
