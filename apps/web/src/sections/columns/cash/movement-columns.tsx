import { useText } from "@pengi/shared";
import type { DataTableColumnDef } from "@pengi/ui";
import { DataTableColumnHeader, Text } from "@pengi/ui";
import { ArrowDownLeft, ArrowUpRight } from "lucide-react";
import type { CashMovement } from "@/api/cash-service";
import { FormattedDate, Money } from "@/components/custom/formatted";
import { Outflow } from "@/components/features/cash/cash-figures";
import {
	movementCategoryLabel,
	movementTypeLabel,
} from "@/components/features/cash/cash-labels";

function MovementLabel({ movement }: { movement: CashMovement }) {
	const { textGet } = useText();
	const Icon = movement.type === "in" ? ArrowDownLeft : ArrowUpRight;
	return (
		<span className="inline-flex items-center gap-1.5">
			<Icon
				aria-hidden
				className={
					movement.type === "in"
						? "h-4 w-4 shrink-0 text-emerald-600 dark:text-emerald-400"
						: "h-4 w-4 shrink-0 text-destructive"
				}
			/>
			{movementCategoryLabel(textGet, movement.category)}
			<span className="text-xs text-muted-foreground">
				({movementTypeLabel(textGet, movement.type)})
			</span>
		</span>
	);
}

/** Columns of a session's manual cash movements. */
export function getMovementColumns(): DataTableColumnDef<CashMovement>[] {
	return [
		{
			accessorKey: "CreatedAt",
			meta: { title: "cash.movement.column.time" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="cash.movement.column.time" />}
				/>
			),
			cell: ({ row }) => (
				<FormattedDate value={row.original.CreatedAt} withTime />
			),
		},
		{
			accessorKey: "category",
			meta: { title: "cash.movement.column.category", phone: "title" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="cash.movement.column.category" />}
				/>
			),
			cell: ({ row }) => <MovementLabel movement={row.original} />,
		},
		{
			accessorKey: "reason",
			meta: { title: "cash.movement.column.reason", phone: "subtitle" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="cash.movement.column.reason" />}
				/>
			),
			cell: ({ row }) => <span>{row.original.reason}</span>,
		},
		{
			accessorKey: "amount",
			meta: { title: "cash.movement.column.amount", phone: "end" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="cash.movement.column.amount" />}
				/>
			),
			cell: ({ row }) => {
				const m = row.original;
				// Outflows read with a minus and a destructive tone (cash-figures).
				return m.type === "out" ? (
					<Outflow amount={m.amount} className="font-medium" />
				) : (
					<Money amount={m.amount} className="font-medium tabular-nums" />
				);
			},
		},
	];
}
