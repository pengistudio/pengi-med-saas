/** biome-ignore-all lint/correctness/useHookAtTopLevel: Only used inside a component */
import type { CellContext, ColumnDef } from "@tanstack/react-table";
import {
	Download,
	Eye,
	MoreVertical,
	Pencil,
	Phone,
	Plus,
	TriangleAlert,
} from "lucide-react";
import { useNavigate } from "react-router";
import type { Patient } from "@/api/clinical-service";
import { downloadPatientReport } from "@/api/clinical-service";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuGroup,
	DropdownMenuItem,
	DropdownMenuLabel,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Text } from "@/components/ui/text";
import useAuth from "@/hooks/use-auth";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS } from "@/lib/constants";
import { usePatientStore } from "@/store/patient-store";

function renderActions({ row }: CellContext<Patient, unknown>) {
	const navigate = useNavigate();
	const { setPatient } = usePatientStore();
	const { token } = useAuth();
	const params = new URLSearchParams({
		patient_id: String(row.original.ID),
	});
	const { checkPermission } = usePermission();
	return (
		<DropdownMenu>
			<DropdownMenuTrigger>
				<Button variant="outline" size="icon" className="ml-auto">
					<MoreVertical className="h-4 w-4" />
					<span className="sr-only">Abrir Menu</span>
				</Button>
			</DropdownMenuTrigger>
			<DropdownMenuContent align="end" className="w-48">
				<DropdownMenuGroup>
					<DropdownMenuLabel>
						<Text uuid="table.actions" />
					</DropdownMenuLabel>
					{checkPermission([
						PERMISSIONS.MEDICAL_RECORD.PERMISSION_CREATE_PATIENT,
					]) && (
						<DropdownMenuItem
							onClick={() => {
								navigate(
									`/clinical/medical-records/create?${params.toString()}`,
								);
							}}
						>
							<Plus className="w-4 h-4 mr-2" />
							<Text uuid="clinical.medical_record.new" />
						</DropdownMenuItem>
					)}
					<DropdownMenuItem
						onClick={() => {
							setPatient(row.original);
							navigate(`/clinical/medical-records/${row.original.ID}`);
						}}
					>
						<Eye className="w-4 h-4 mr-2" />
						<Text uuid="clinical.medical_record.view" />
					</DropdownMenuItem>
					{checkPermission([
						PERMISSIONS.MEDICAL_RECORD.PERMISSION_UPDATE_PATIENT,
					]) && (
						<DropdownMenuItem
							onClick={() => {
								navigate(`/clinical/edit/${row.original.ID}`);
							}}
						>
							<Pencil className="w-4 h-4 mr-2" />
							<Text uuid="clinical.patient.edit" />
						</DropdownMenuItem>
					)}
					{checkPermission([
						PERMISSIONS.MEDICAL_RECORD.PERMISSION_DOWNLOAD_PATIENT_REPORT,
					]) && (
						<DropdownMenuItem
							onClick={async () => {
								if (token) {
									await downloadPatientReport(row.original.ID);
								}
							}}
						>
							<Download className="w-4 h-4 mr-2 mb-1" />
							<Text uuid="clinical.patient.download_records" />
						</DropdownMenuItem>
					)}
				</DropdownMenuGroup>
			</DropdownMenuContent>
		</DropdownMenu>
	);
}

export const patientColumns: ColumnDef<Patient>[] = [
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
		accessorKey: "last_name",
		header: () => <Text uuid="clinical.patient.last_name" />,
		meta: { title: "table.column.last_name" },
		size: 150,
	},
	{
		accessorKey: "first_name",
		header: () => <Text uuid="clinical.patient.first_name" />,
		meta: { title: "table.column.first_name" },
		size: 150,
	},
	{
		accessorKey: "document",
		header: () => <Text uuid="clinical.patient.document" />,
		meta: { title: "table.column.document" },
		size: 50,
	},

	{
		accessorKey: "phone",
		header: () => <Text uuid="clinical.patient.phone" />,
		meta: { title: "table.column.phone" },
		cell: ({ row }) => {
			return (
				<div className="flex items-center gap-2">
					{row.original.phone ? (
						<span>{row.original.phone}</span>
					) : (
						<Badge variant="secondary" className="font-normal">
							<Phone className="h-3 w-3 mr-1" />
							<Text uuid="clinical.patient.phone.unregistered" />
						</Badge>
					)}
				</div>
			);
		},
		size: 50,
	},
	{
		accessorKey: "diagnosis",
		header: () => <Text uuid="clinical.patient.diagnosis" />,
		meta: { title: "table.column.diagnosis" },
		cell: ({ row }) => {
			return (
				<div className="flex items-center gap-2">
					{row.original.diagnosis ? (
						<span>{row.original.diagnosis}</span>
					) : (
						<Badge variant="secondary" className="font-normal">
							<Text uuid="clinical.patient.diagnosis.none" />
						</Badge>
					)}
				</div>
			);
		},
		size: 100,
	},
	{
		id: "next_appointment",
		header: () => <Text uuid="clinical.patient.next_appointment" />,
		meta: { title: "table.column.next_appointment" },
		size: 100,
		cell: ({ row }) => {
			const medicalRecords = row.original.medical_records;
			if (!medicalRecords || medicalRecords.length === 0) {
				return (
					<Badge variant="secondary" className="font-normal">
						<Text uuid="clinical.patient.appointment.none" />
					</Badge>
				);
			}
			const nextAppointmentDate = medicalRecords[0]?.next_appointment_date;
			if (!nextAppointmentDate) {
				return (
					<Badge variant="secondary" className="font-normal">
						<Text uuid="clinical.patient.appointment.none" />
					</Badge>
				);
			}
			return new Date(nextAppointmentDate as string).toLocaleDateString(
				"es-EC",
				{
					year: "numeric",
					month: "short",
					day: "numeric",
				},
			);
		},
	},
	{
		accessorKey: "medic",
		header: () => <Text uuid="clinical.patient.medic" />,
		meta: { title: "table.column.medic" },
		size: 50,
	},
	{
		cell: renderActions,
		id: "actions",
		size: 50,
	},
];
export const patientColumnsMobile: ColumnDef<Patient>[] = [
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
		id: "warning",
		size: 50,
		cell: ({ row }) => {
			if (!row.original.critical) return null;
			return <TriangleAlert className="text-destructive w-4 h-4" />;
		},
	},
	{
		accessorKey: "full_name",
		header: () => <Text uuid="clinical.patient.name" />,
	},
	{
		cell: renderActions,
		id: "actions",
		size: 50,
	},
];
