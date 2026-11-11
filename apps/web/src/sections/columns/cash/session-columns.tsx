import { useText } from "@pengi/shared";
import type { DataTableColumnDef } from "@pengi/ui";
import { Button, DataTableColumnHeader, Text } from "@pengi/ui";
import { Eye } from "lucide-react";
import { useNavigate } from "react-router";
import type { CashSession } from "@/api/cash-service";
import { FormattedDate, Money } from "@/components/custom/formatted";
import { DifferenceBadge } from "@/components/features/cash/cash-figures";

function OpenSessionButton({ id }: { id: number }) {
	const navigate = useNavigate();
	const { textGet } = useText();
	return (
		<Button
			variant="ghost"
			size="sm"
			onClick={() => navigate(`/cash/sessions/${id}`)}
		>
			<Eye className="h-4 w-4" />
			{textGet("cash.history.view")}
		</Button>
	);
}

/** Columns of the cash session history. */
export function getSessionColumns(): DataTableColumnDef<CashSession>[] {
	return [
		{
			accessorKey: "opened_at",
			meta: { title: "cash.history.column.opened_at", phone: "title" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="cash.history.column.opened_at" />}
				/>
			),
			cell: ({ row }) => (
				<FormattedDate value={row.original.opened_at} withTime />
			),
		},
		{
			accessorKey: "closed_at",
			meta: { title: "cash.history.column.closed_at" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="cash.history.column.closed_at" />}
				/>
			),
			cell: ({ row }) => (
				<FormattedDate value={row.original.closed_at} withTime />
			),
		},
		{
			accessorKey: "closed_by_name",
			meta: { title: "cash.history.column.closed_by", phone: "subtitle" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="cash.history.column.closed_by" />}
				/>
			),
			cell: ({ row }) => <span>{row.original.closed_by_name || "—"}</span>,
		},
		{
			accessorKey: "expected_cash",
			meta: { title: "cash.history.column.expected" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="cash.history.column.expected" />}
				/>
			),
			cell: ({ row }) => (
				<Money amount={row.original.expected_cash} className="tabular-nums" />
			),
		},
		{
			accessorKey: "counted_cash",
			meta: { title: "cash.history.column.counted" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="cash.history.column.counted" />}
				/>
			),
			cell: ({ row }) => (
				<Money amount={row.original.counted_cash} className="tabular-nums" />
			),
		},
		{
			accessorKey: "difference",
			meta: { title: "cash.history.column.difference", phone: "end" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="cash.history.column.difference" />}
				/>
			),
			cell: ({ row }) => <DifferenceBadge value={row.original.difference} />,
		},
		{
			id: "actions",
			header: () => null,
			cell: ({ row }) => <OpenSessionButton id={row.original.ID} />,
			enableSorting: false,
			enableHiding: false,
		},
	];
}
