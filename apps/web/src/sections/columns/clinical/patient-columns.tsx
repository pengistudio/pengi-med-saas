/** biome-ignore-all lint/correctness/useHookAtTopLevel: Only used inside a component */
import type { CellContext, ColumnDef } from "@tanstack/react-table";
import {
	Download,
	Eye,
	HelpCircle,
	MoreVertical,
	Pencil,
	Plus,
} from "lucide-react";
import { useNavigate } from "react-router";
import type { Patient } from "@/api/clinical-service";
import { downloadPatientReport } from "@/api/clinical-service";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
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
import { cn } from "@/lib/utils";
import { usePatientStore } from "@/store/patient-store";

function getInitials(patient: Patient): string {
	const first = patient.first_name?.[0] ?? "";
	const last = patient.last_name?.[0] ?? "";
	return (first + last).toUpperCase();
}

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
			<DropdownMenu>
				<DropdownMenuTrigger
					render={
						<Button variant="ghost" size="icon">
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
		id: "patient",
		header: () => <Text uuid="clinical.patient.name" />,
		meta: { title: "table.column.patient" },
		size: 220,
		cell: ({ row }) => {
			const p = row.original;
			const fullName = `${p.last_name} ${p.first_name}`.trim();
			return (
				<div className="flex items-center gap-3">
					<TooltipProvider>
						<Tooltip>
							<TooltipTrigger>
								<Avatar
									className={cn(
										"h-8 w-8 shrink-0 text-xs",
										p.critical ? "ring-2 ring-destructive" : "",
									)}
								>
									<AvatarFallback
										className={
											p.critical ? "bg-destructive/10 text-destructive" : ""
										}
									>
										{getInitials(p)}
									</AvatarFallback>
								</Avatar>
							</TooltipTrigger>
							{p.critical && (
								<TooltipContent side="right">
									<Text uuid="clinical.patient.critical.label" />
								</TooltipContent>
							)}
						</Tooltip>
					</TooltipProvider>
					<span className="font-medium truncate">{fullName}</span>
				</div>
			);
		},
	},
	{
		accessorKey: "document",
		header: () => <Text uuid="clinical.patient.document" />,
		meta: { title: "table.column.document" },
		size: 130,
	},
	{
		accessorKey: "phone",
		header: () => <Text uuid="clinical.patient.phone" />,
		meta: { title: "table.column.phone" },
		cell: ({ row }) => {
			return row.original.phone ? (
				<span>{row.original.phone}</span>
			) : (
				<span className="text-muted-foreground">—</span>
			);
		},
		size: 120,
	},
	{
		accessorKey: "diagnosis",
		header: () => <Text uuid="clinical.patient.diagnosis" />,
		meta: { title: "table.column.diagnosis" },
		cell: ({ row }) => {
			return row.original.diagnosis ? (
				<span className="truncate max-w-[180px] block">
					{row.original.diagnosis}
				</span>
			) : (
				<span className="text-muted-foreground">—</span>
			);
		},
		size: 180,
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
							<div className="flex items-center gap-1.5">
								<span className="inline-block h-2 w-2 rounded-full bg-blue-500" />
								<span>
									<Text uuid="clinical.patient.appointment.suggested" />
								</span>
							</div>
						</TooltipContent>
					</Tooltip>
				</TooltipProvider>
			</div>
		),
		meta: { title: "table.column.next_appointment" },
		size: 120,
		cell: ({ row }) => {
			const appointments = row.original.appointments;
			const nextScheduled = appointments?.find((a) => a.status === "scheduled");

			// Prioridad 1: cita formal agendada
			if (nextScheduled) {
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
			}

			// Prioridad 2: fecha sugerida por el médico
			const lastRecord = row.original.medical_records?.[0];
			if (lastRecord?.next_appointment_date) {
				const suggestedDate = new Date(lastRecord.next_appointment_date);
				const dateLabel = suggestedDate.toLocaleDateString("es-EC", {
					month: "short",
					day: "numeric",
				});
				return (
					<span className="inline-flex flex-col gap-0.5 rounded-md border px-2 py-1 text-xs font-medium bg-blue-500/15 text-blue-700 dark:text-blue-400 border-blue-400/40">
						<span>{dateLabel}</span>
						<span className="opacity-80 text-[10px]">
							<Text uuid="clinical.patient.appointment.suggested" />
						</span>
					</span>
				);
			}

			return <span className="text-muted-foreground">—</span>;
		},
	},
	{
		accessorKey: "medic",
		header: () => <Text uuid="clinical.patient.medic" />,
		meta: { title: "table.column.medic" },
		cell: ({ row }) =>
			row.original.medic ? (
				<span>{row.original.medic}</span>
			) : (
				<span className="text-muted-foreground">—</span>
			),
		size: 120,
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
		accessorKey: "full_name",
		header: () => <Text uuid="clinical.patient.name" />,
	},
	{
		cell: renderActions,
		id: "actions",
		size: 50,
	},
];
