import { DataTableColumnHeader, Text } from "@pengi/ui";
import type { ColumnDef } from "@tanstack/react-table";
import { Link } from "react-router";
import type { ExamOrder } from "@/api/exam-order-service";
import { FormattedDate } from "@/components/custom/formatted";
import {
	ExamOrderStatusBadge,
	PendingReviewBadge,
} from "@/components/features/exam-orders/exam-order-status-badge";
import { patientDisplayName } from "@/lib/exam-orders";

interface ExamOrderColumnOptions {
	/** Hide the patient column (lists already scoped to one patient). */
	showPatient?: boolean;
}

export const getExamOrderColumns = ({
	showPatient = true,
}: ExamOrderColumnOptions = {}): ColumnDef<ExamOrder>[] => {
	const columns: ColumnDef<ExamOrder>[] = [
		{
			accessorKey: "code",
			meta: { title: "clinical.exam_orders.column.code", phone: "title" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="clinical.exam_orders.column.code" />}
				/>
			),
			cell: ({ row }) => (
				<Link
					to={`/clinical/exam-orders/${row.original.ID}`}
					className="font-medium text-primary underline-offset-4 hover:underline"
				>
					{row.original.code}
				</Link>
			),
		},
	];
	if (showPatient) {
		columns.push({
			id: "patient",
			meta: { title: "clinical.exam_orders.column.patient", phone: "subtitle" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="clinical.exam_orders.column.patient" />}
				/>
			),
			cell: ({ row }) => (
				<span>{patientDisplayName(row.original.patient) || "—"}</span>
			),
		});
	}
	columns.push(
		{
			accessorKey: "CreatedAt",
			meta: { title: "clinical.exam_orders.column.date", phone: "end" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="clinical.exam_orders.column.date" />}
				/>
			),
			cell: ({ row }) => <FormattedDate value={row.original.CreatedAt} />,
		},
		{
			accessorKey: "status",
			meta: { title: "clinical.exam_orders.column.status", phone: "status" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="clinical.exam_orders.column.status" />}
				/>
			),
			cell: ({ row }) => (
				<span className="flex flex-wrap items-center gap-1">
					<ExamOrderStatusBadge status={row.original.status} />
					{row.original.pending_review && <PendingReviewBadge />}
				</span>
			),
		},
		{
			id: "exams",
			meta: { title: "clinical.exam_orders.column.exams" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="clinical.exam_orders.column.exams" />}
				/>
			),
			cell: ({ row }) => (
				<span className="tabular-nums">{row.original.items?.length ?? 0}</span>
			),
		},
		{
			accessorKey: "ordered_by_name",
			meta: { title: "clinical.exam_orders.column.ordered_by" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="clinical.exam_orders.column.ordered_by" />}
				/>
			),
			cell: ({ row }) => <span>{row.original.ordered_by_name || "—"}</span>,
		},
	);
	return columns;
};
