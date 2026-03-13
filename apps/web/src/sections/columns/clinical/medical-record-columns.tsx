import type { CellContext, ColumnDef } from "@tanstack/react-table";
import {
	Activity,
	CopyPlus,
	Download,
	Edit,
	Eye,
	FileSearch,
	MoreVertical,
	Pill,
	Plus,
} from "lucide-react";
import type { MedicalRecord } from "@/api/clinical-service";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuGroup,
	DropdownMenuItem,
	DropdownMenuLabel,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Text } from "@/components/ui/text";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS } from "@/lib/constants";
import { dateParser } from "@/lib/utils";

interface ActionCellProps {
	row: CellContext<MedicalRecord, unknown>["row"];
	onView?: (id: number) => void;
	onEdit?: (id: number) => void;
	onViewPrescription?: (record: MedicalRecord) => void;
	onEditPrescription?: (record: MedicalRecord) => void;
	onDownloadPrescription?: (record: MedicalRecord) => void;
}

function ActionsCell({
	row,
	onView,
	onEdit,
	onViewPrescription,
	onEditPrescription,
	onDownloadPrescription,
}: ActionCellProps) {
	const { checkPermission } = usePermission();
	const hasPrescription = !!row.original.prescription;

	return (
		<DropdownMenu>
			<DropdownMenuTrigger>
				<Button variant="outline" size="icon" className="ml-auto">
					<MoreVertical className="h-4 w-4" />
					<span className="sr-only">Abrir Menu</span>
				</Button>
			</DropdownMenuTrigger>
			<DropdownMenuContent align="end" className="w-64">
				<DropdownMenuGroup>
					<DropdownMenuLabel>
						<Text uuid="table.actions" />
					</DropdownMenuLabel>
					{checkPermission([
						PERMISSIONS.MEDICAL_RECORD.PERMISSION_READ_MEDICAL_RECORD,
					]) && (
						<DropdownMenuItem onClick={() => onView?.(row.original.ID)}>
							<Eye className="w-4 h-4 mr-2" />
							<Text uuid="clinical.medical_record.view" />
						</DropdownMenuItem>
					)}
					{checkPermission([
						PERMISSIONS.MEDICAL_RECORD.PERMISSION_UPDATE_MEDICAL_RECORD,
					]) && (
						<DropdownMenuItem onClick={() => onEdit?.(row.original.ID)}>
							<Edit className="w-4 h-4 mr-2" />
							<Text uuid="clinical.medical_record.update" />
						</DropdownMenuItem>
					)}
				</DropdownMenuGroup>
				<DropdownMenuSeparator />
				<DropdownMenuGroup>
					<DropdownMenuLabel>
						<Text uuid="clinical.medical_record.prescription" />
					</DropdownMenuLabel>
					{hasPrescription ? (
						<>
							<DropdownMenuItem
								onClick={() => onViewPrescription?.(row.original)}
							>
								<FileSearch className="w-4 h-4 mr-2" />
								<Text uuid="clinical.medical_record.view_prescription" />
							</DropdownMenuItem>
							{checkPermission([
								PERMISSIONS.MEDICAL_RECORD.PERMISSION_UPDATE_MEDICAL_RECORD,
							]) && (
								<DropdownMenuItem
									onClick={() => onEditPrescription?.(row.original)}
								>
									<Pill className="w-4 h-4 mr-2" />
									<Text uuid="clinical.medical_record.edit_prescription" />
								</DropdownMenuItem>
							)}
							<DropdownMenuItem
								onClick={() => onDownloadPrescription?.(row.original)}
							>
								<Download className="w-4 h-4 mr-2" />
								<Text uuid="view.medical_record.prescription.download" />
							</DropdownMenuItem>
						</>
					) : (
						checkPermission([
							PERMISSIONS.MEDICAL_RECORD.PERMISSION_UPDATE_MEDICAL_RECORD,
						]) && (
							<DropdownMenuItem
								onClick={() => onEditPrescription?.(row.original)}
							>
								<Plus className="w-4 h-4 mr-2" />
								<Text uuid="clinical.medical_record.create_prescription" />
							</DropdownMenuItem>
						)
					)}
				</DropdownMenuGroup>
			</DropdownMenuContent>
		</DropdownMenu>
	);
}

export const getMedicalRecordColumns = (
	onView?: (id: number) => void,
	onEdit?: (id: number) => void,
	onViewPrescription?: (record: MedicalRecord) => void,
	onEditPrescription?: (record: MedicalRecord) => void,
	onDownloadPrescription?: (record: MedicalRecord) => void,
): ColumnDef<MedicalRecord>[] => [
	{
		cell: ({ row }) => (
			<Checkbox
				aria-label="Select row"
				checked={row.getIsSelected()}
				onCheckedChange={(value) => row.toggleSelected(!!value)}
				onClick={(e) => e.stopPropagation()}
			/>
		),
		enableHiding: false,
		enableSorting: false,
		header: ({ table }) => (
			<Checkbox
				aria-label="Select all"
				checked={table.getIsAllPageRowsSelected()}
				indeterminate={table.getIsSomePageRowsSelected()}
				onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
			/>
		),
		id: "select",
		size: 50,
	},
	{
		accessorFn: (row) =>
			dateParser(new Date(row.date), { dateStyle: "medium" }),
		id: "date",
		header: () => <Text uuid="clinical.medical_record.date" />,
		meta: { title: "table.column.date" },
		size: 150,
	},
	{
		accessorKey: "motive",
		header: () => <Text uuid="clinical.medical_record.motive" />,
		meta: { title: "table.column.motive" },
		size: 200,
	},
	{
		accessorKey: "observation",
		header: () => <Text uuid="clinical.medical_record.observation" />,
		meta: { title: "table.column.observation" },
		size: 250,
		cell: ({ row }) => {
			const obs = row.original.observation;
			return (
				<span className="line-clamp-2" title={obs}>
					{obs}
				</span>
			);
		},
	},
	{
		id: "vital_signs",
		header: () => <Text uuid="clinical.medical_record.vital_signs" />,
		meta: { title: "table.column.vital_signs" },
		size: 80,
		cell: ({ row }) => {
			return row.original.vital_signs ? (
				<Activity className="w-5 h-5 text-rose-500" />
			) : (
				<span className="text-muted-foreground">-</span>
			);
		},
	},
	{
		id: "prescription",
		header: () => <Text uuid="clinical.medical_record.prescription" />,
		meta: { title: "table.column.prescription" },
		size: 100,
		cell: ({ row }) => {
			return row.original.prescription ? (
				<CopyPlus className="w-5 h-5 text-primary" />
			) : (
				<span className="text-muted-foreground">-</span>
			);
		},
	},
	{
		id: "actions",
		cell: ({ row }) => (
			<ActionsCell
				row={row}
				onView={onView}
				onEdit={onEdit}
				onViewPrescription={onViewPrescription}
				onEditPrescription={onEditPrescription}
				onDownloadPrescription={onDownloadPrescription}
			/>
		),
		size: 50,
	},
];
