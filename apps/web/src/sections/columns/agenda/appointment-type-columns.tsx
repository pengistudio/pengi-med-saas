import { useText } from "@pengi/shared";
import {
	Button,
	DataTableColumnHeader,
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuGroup,
	DropdownMenuItem,
	DropdownMenuTrigger,
	Text,
} from "@pengi/ui";
import type { ColumnDef } from "@tanstack/react-table";
import { MoreHorizontal, Pencil, Trash2 } from "lucide-react";
import type { AppointmentType } from "@/api/agenda-service";
import type { Doctor } from "@/api/doctors-service";
import { Switch } from "@/components/ui/switch";
import { findDoctor } from "@/store/doctors-store";

interface TypeActions {
	onEdit: (row: AppointmentType) => void;
	onDelete: (row: AppointmentType) => void;
	onToggleActive: (row: AppointmentType, active: boolean) => void;
}

function DurationCell({ minutes }: { minutes: number }) {
	const { textGet } = useText();
	return <span>{textGet("agenda.types.duration_value", { minutes })}</span>;
}

function DoctorsCell({
	type,
	doctors,
}: {
	type: AppointmentType;
	doctors: Doctor[];
}) {
	const { textGet } = useText();
	const ids = type.doctor_ids ?? [];
	if (ids.length === 0) {
		return (
			<span className="text-muted-foreground">
				{textGet("agenda.types.all_doctors")}
			</span>
		);
	}
	const names = ids
		.map((id) => findDoctor(doctors, id)?.full_name)
		.filter(Boolean)
		.join(", ");
	return (
		<span className="block max-w-xs truncate text-sm" title={names}>
			{names}
		</span>
	);
}

export const getAppointmentTypeColumns = (
	actions: TypeActions,
	doctors: Doctor[],
): ColumnDef<AppointmentType>[] => [
	{
		accessorKey: "name",
		meta: { title: "agenda.types.field.name", phone: "title" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="agenda.types.field.name" />}
			/>
		),
		cell: ({ row }) => (
			<span className="flex items-center gap-2 font-medium">
				{row.original.color && (
					<span
						aria-hidden
						className="size-2.5 shrink-0 rounded-full"
						style={{ backgroundColor: row.original.color }}
					/>
				)}
				{row.original.name}
			</span>
		),
	},
	{
		accessorKey: "duration_minutes",
		meta: { title: "agenda.types.field.duration", phone: "subtitle" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="agenda.types.field.duration" />}
			/>
		),
		cell: ({ row }) => <DurationCell minutes={row.original.duration_minutes} />,
	},
	{
		id: "doctors",
		meta: { title: "agenda.types.field.doctors" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="agenda.types.field.doctors" />}
			/>
		),
		cell: ({ row }) => <DoctorsCell type={row.original} doctors={doctors} />,
	},
	{
		accessorKey: "active",
		meta: { title: "agenda.types.field.active", phone: "end" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="agenda.types.field.active" />}
			/>
		),
		cell: ({ row }) => (
			<Switch
				checked={row.original.active}
				onCheckedChange={(checked) =>
					actions.onToggleActive(row.original, checked)
				}
			/>
		),
	},
	{
		id: "actions",
		cell: ({ row }) => (
			<DropdownMenu>
				<DropdownMenuTrigger
					render={
						<Button variant="ghost" size="icon">
							<Text uuid="table.button.open_menu" className="sr-only" />
							<MoreHorizontal className="h-4 w-4" />
						</Button>
					}
				/>
				<DropdownMenuContent align="end">
					<DropdownMenuGroup>
						<DropdownMenuItem onClick={() => actions.onEdit(row.original)}>
							<Pencil className="mr-2 h-4 w-4" />
							<Text uuid="agenda.types.edit" />
						</DropdownMenuItem>
						<DropdownMenuItem
							className="text-destructive"
							onClick={() => actions.onDelete(row.original)}
						>
							<Trash2 className="mr-2 h-4 w-4" />
							<Text uuid="agenda.types.delete" />
						</DropdownMenuItem>
					</DropdownMenuGroup>
				</DropdownMenuContent>
			</DropdownMenu>
		),
	},
];
