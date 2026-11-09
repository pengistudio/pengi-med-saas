import { useText } from "@pengi/shared";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@pengi/ui";
import { ArrowLeftRight, ReceiptText } from "lucide-react";
import { useState } from "react";
import type { CashMovement, PaymentView } from "@/api/cash-service";
import { DataTable } from "@/components/custom/table/data-table";
import type { getMovementColumns } from "@/sections/columns/cash/movement-columns";
import type { getPaymentColumns } from "@/sections/columns/cash/payment-columns";
import { CashEmptyState } from "./cash-figures";

type Tab = "payments" | "movements";

function Count({ value }: { value: number }) {
	return (
		<span className="rounded-full bg-foreground/10 px-1.5 text-xs tabular-nums">
			{value}
		</span>
	);
}

/**
 * A session's payments and movements, one tab each (payments first: what the
 * cashier looks at most). The empty states say how each list gets filled.
 */
export function SessionActivity({
	payments,
	movements,
	paymentColumns,
	movementColumns,
	paymentsHint = false,
	movementsHint = false,
}: {
	payments: PaymentView[];
	movements: CashMovement[];
	paymentColumns: ReturnType<typeof getPaymentColumns>;
	movementColumns: ReturnType<typeof getMovementColumns>;
	/** The open session: say where payments come from. */
	paymentsHint?: boolean;
	/** The user may record movements: point to the button. */
	movementsHint?: boolean;
}) {
	const { textGet } = useText();
	const [tab, setTab] = useState<Tab>("payments");

	return (
		<Tabs value={tab} onValueChange={(v) => setTab(v as Tab)}>
			<TabsList className="w-full sm:w-fit">
				<TabsTrigger value="payments" className="flex-1 gap-1.5 sm:px-3">
					<ReceiptText aria-hidden className="h-3.5 w-3.5" />
					{textGet("cash.session.payments")}
					<Count value={payments.length} />
				</TabsTrigger>
				<TabsTrigger value="movements" className="flex-1 gap-1.5 sm:px-3">
					<ArrowLeftRight aria-hidden className="h-3.5 w-3.5" />
					{textGet("cash.session.movements")}
					<Count value={movements.length} />
				</TabsTrigger>
			</TabsList>
			<TabsContent value="payments">
				<DataTable
					columns={paymentColumns}
					data={payments}
					emptyState={
						<CashEmptyState
							icon={ReceiptText}
							title={textGet("cash.session.payments.empty")}
							hint={
								paymentsHint
									? textGet("cash.session.payments.empty.hint")
									: undefined
							}
						/>
					}
				/>
			</TabsContent>
			<TabsContent value="movements">
				<DataTable
					columns={movementColumns}
					data={movements}
					emptyState={
						<CashEmptyState
							icon={ArrowLeftRight}
							title={textGet("cash.session.movements.empty")}
							hint={
								movementsHint
									? textGet("cash.session.movements.empty.hint")
									: undefined
							}
						/>
					}
				/>
			</TabsContent>
		</Tabs>
	);
}
