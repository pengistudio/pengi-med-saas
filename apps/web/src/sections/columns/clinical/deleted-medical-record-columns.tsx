import { Button, Text } from "@pengi/ui";
import type { ColumnDef } from "@tanstack/react-table";
import { ArchiveRestore, Loader2 } from "lucide-react";
import type { DeletedMedicalRecord } from "@/api/clinical-service";
import { FormattedDate } from "@/components/custom/formatted";

/** A patient's deleted medical records: what, who, when, why, and restore. */
export const getDeletedMedicalRecordColumns = (
	onRestore: (record: DeletedMedicalRecord) => void,
	restoringId: number | null,
): ColumnDef<DeletedMedicalRecord>[] => [
	{
		id: "date",
		accessorFn: (row) => row.date,
		cell: ({ row }) => <FormattedDate value={row.original.date} />,
		header: () => <Text uuid="clinical.medical_record.date" />,
		meta: { title: "table.column.date", phone: "title" },
		size: 130,
	},
	{
		accessorKey: "motive",
		header: () => <Text uuid="clinical.medical_record.motive" />,
		meta: { title: "table.column.motive", phone: "subtitle" },
		size: 200,
	},
	{
		id: "deleted_by",
		accessorFn: (row) => row.deleted_by_name ?? "",
		header: () => <Text uuid="clinical.medical_record.deleted.column.by" />,
		meta: { title: "clinical.medical_record.deleted.column.by" },
		size: 140,
	},
	{
		id: "deleted_at",
		accessorFn: (row) => row.DeletedAt,
		cell: ({ row }) =>
			row.original.DeletedAt ? (
				<FormattedDate value={row.original.DeletedAt} withTime />
			) : null,
		header: () => <Text uuid="clinical.medical_record.deleted.column.at" />,
		meta: { title: "clinical.medical_record.deleted.column.at" },
		size: 170,
	},
	{
		accessorKey: "delete_reason",
		header: () => <Text uuid="clinical.medical_record.deleted.column.reason" />,
		meta: { title: "clinical.medical_record.deleted.column.reason" },
		size: 250,
		cell: ({ row }) => (
			<span className="line-clamp-3 text-sm" title={row.original.delete_reason}>
				{row.original.delete_reason}
			</span>
		),
	},
	{
		id: "actions",
		cell: ({ row }) => (
			<Button
				type="button"
				variant="outline"
				size="sm"
				className="ml-auto"
				disabled={restoringId === row.original.ID}
				onClick={() => onRestore(row.original)}
			>
				{restoringId === row.original.ID ? (
					<Loader2 className="mr-2 h-4 w-4 animate-spin" />
				) : (
					<ArchiveRestore className="mr-2 h-4 w-4" />
				)}
				<Text uuid="clinical.medical_record.restore" />
			</Button>
		),
		size: 130,
	},
];
