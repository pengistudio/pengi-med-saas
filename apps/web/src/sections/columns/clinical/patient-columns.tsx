/** biome-ignore-all lint/correctness/useHookAtTopLevel: Only used inside a component */
import type { CellContext, ColumnDef } from "@tanstack/react-table";
import {
	Download,
	Eye,
	HelpCircle,
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
import {
	Tooltip,
	TooltipContent,
	TooltipProvider,
	TooltipTrigger,
} from "@/components/ui/tooltip";
import useAuth from "@/hooks/use-auth";
import usePermission from "@/hooks/use-permission";
import useTenantSettings from "@/hooks/use-tenant-settings";
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
		<div className="flex items-center gap-1 justify-end">
			{row.original.critical && (
				<TooltipProvider>
					<Tooltip>
						<TooltipTrigger className="cursor-default">
							<TriangleAlert className="text-destructive w-4 h-4 shrink-0" />
						</TooltipTrigger>
						<TooltipContent side="left">
							<Text uuid="clinical.patient.critical.label" />
						</TooltipContent>
					</Tooltip>
				</TooltipProvider>
			)}
			<DropdownMenu>
			<DropdownMenuTrigger
				render={
					<Button variant="outline" size="icon">
						<MoreVertical className="h-4 w-4" />
						<span className="sr-only">Abrir Menu</span>
					</Button>
				}
			/>
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
		</div>
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
		header: () => (
			<div className="flex items-center gap-1">
				<Text uuid="clinical.patient.next_appointment" />
				<TooltipProvider>
					<Tooltip>
						<TooltipTrigger className="cursor-help">
							<HelpCircle className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
						</TooltipTrigger>
						<TooltipContent side="top" className="max-w-48 text-xs space-y-1">
							<p className="font-medium">
								<Text uuid="clinical.patient.next_appointment" />
							</p>
							<div className="flex items-center gap-1.5">
								<span className="inline-block h-2 w-2 rounded-full bg-red-500" />
								<span>
									<Text uuid="clinical.patient.appointment.today" />
								</span>
							</div>
							<div className="flex items-center gap-1.5">
								<span className="inline-block h-2 w-2 rounded-full bg-amber-500" />
								<span>
									<Text uuid="clinical.patient.appointment.tomorrow" />
								</span>
							</div>
							<div className="flex items-center gap-1.5">
								<span className="inline-block h-2 w-2 rounded-full bg-emerald-500" />
								<span>
									<Text uuid="clinical.patient.appointment.upcoming" />
								</span>
							</div>
						</TooltipContent>
					</Tooltip>
				</TooltipProvider>
			</div>
		),
		meta: { title: "table.column.next_appointment" },
		size: 100,
		cell: ({ row }) => {
			const appointments = row.original.appointments;
			const nextScheduled = appointments?.find((a) => a.status === "scheduled");
			if (!nextScheduled) {
				return (
					<Badge variant="secondary" className="font-normal">
						<Text uuid="clinical.patient.appointment.none" />
					</Badge>
				);
			}

			const apptDate = new Date(nextScheduled.date);
			const today = new Date();
			today.setHours(0, 0, 0, 0);
			const apptDay = new Date(apptDate);
			apptDay.setHours(0, 0, 0, 0);
			const diffDays = Math.round(
				(apptDay.getTime() - today.getTime()) / 86400000,
			);

			const colorClass =
				diffDays === 0
					? "bg-red-500/15 text-red-700 dark:text-red-400 border-red-400/40"
					: diffDays === 1
						? "bg-amber-500/15 text-amber-700 dark:text-amber-400 border-amber-400/40"
						: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-400 border-emerald-400/40";

			const dateLabel = apptDate.toLocaleDateString("es-EC", {
				month: "short",
				day: "numeric",
			});

			return (
				<span
					className={`inline-flex flex-col gap-0.5 rounded-md border px-2 py-1 text-xs font-medium ${colorClass}`}
				>
					<span>{dateLabel}</span>
					<span className="opacity-80">
						{nextScheduled.start_time} – {nextScheduled.end_time}
					</span>
				</span>
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
const OPTIONAL_COLUMN_KEYS: Record<string, string> = {
	diagnosis: "show_diagnosis",
	next_appointment: "show_next_appointment",
	medic: "show_medic",
};

export function usePatientColumns(): ColumnDef<Patient>[] {
	const { settings } = useTenantSettings();
	const clinical = settings.clinical;

	return patientColumns.filter((col) => {
		const key =
			(col as { accessorKey?: string }).accessorKey ??
			(col as { id?: string }).id;
		const settingKey = key ? OPTIONAL_COLUMN_KEYS[key] : undefined;
		if (!settingKey) return true;
		return clinical[settingKey as keyof typeof clinical] !== false;
	});
}

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
