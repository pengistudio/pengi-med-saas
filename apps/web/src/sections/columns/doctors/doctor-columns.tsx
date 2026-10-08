import {
	Badge,
	Button,
	DataTableColumnHeader,
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuGroup,
	DropdownMenuItem,
	DropdownMenuLabel,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
	Text,
} from "@pengi/ui";
import type { ColumnDef } from "@tanstack/react-table";
import {
	Link2,
	MoreHorizontal,
	Pencil,
	Power,
	PowerOff,
	Trash2,
} from "lucide-react";
import type { Doctor } from "@/api/doctors-service";
import { specialtyLabel } from "@/components/features/doctors/doctor-utils";

interface DoctorColumnProps {
	textGet: (key: string) => string;
	/** user_id → display name of the linked team member. */
	userNames: Map<number, string>;
	onEdit: (doctor: Doctor) => void;
	onLink: (doctor: Doctor) => void;
	onToggleActive: (doctor: Doctor) => void;
	onDelete: (doctor: Doctor) => void;
}

export const getDoctorColumns = ({
	textGet,
	userNames,
	onEdit,
	onLink,
	onToggleActive,
	onDelete,
}: DoctorColumnProps): ColumnDef<Doctor>[] => [
	{
		accessorKey: "full_name",
		meta: { title: "doctors.column.name", phone: "title" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="doctors.column.name" />}
			/>
		),
		cell: ({ row }) => (
			<span className="flex items-center gap-2 font-medium">
				<span
					aria-hidden
					className="h-3 w-3 shrink-0 rounded-full"
					style={{ backgroundColor: row.original.color }}
				/>
				{row.original.full_name}
				{row.original.needs_review && (
					<Badge
						variant="outline"
						className="border-amber-500/50 text-amber-700"
					>
						<Text uuid="doctors.badge.needs_review" />
					</Badge>
				)}
			</span>
		),
	},
	{
		accessorKey: "specialty",
		meta: { title: "doctors.column.specialty", phone: "subtitle" },
		header: () => <Text uuid="doctors.column.specialty" />,
		cell: ({ row }) => <span>{specialtyLabel(row.original, textGet)}</span>,
	},
	{
		accessorKey: "professional_registry",
		meta: { title: "doctors.column.registry" },
		header: () => <Text uuid="doctors.column.registry" />,
		cell: ({ row }) =>
			row.original.professional_registry ? (
				<span>{row.original.professional_registry}</span>
			) : (
				<span className="text-muted-foreground">—</span>
			),
	},
	{
		accessorKey: "user_id",
		meta: { title: "doctors.column.user" },
		header: () => <Text uuid="doctors.column.user" />,
		cell: ({ row }) => {
			const userId = row.original.user_id;
			if (!userId) {
				return (
					<span className="text-muted-foreground">
						<Text uuid="doctors.user.none" />
					</span>
				);
			}
			return (
				<span>{userNames.get(userId) ?? textGet("doctors.user.linked")}</span>
			);
		},
	},
	{
		accessorKey: "active",
		meta: { title: "doctors.column.status", phone: "end" },
		header: () => <Text uuid="doctors.column.status" />,
		cell: ({ row }) =>
			row.original.active ? (
				<Badge variant="secondary">
					<Text uuid="doctors.status.active" />
				</Badge>
			) : (
				<Badge variant="outline" className="text-muted-foreground">
					<Text uuid="doctors.status.inactive" />
				</Badge>
			),
	},
	{
		id: "actions",
		cell: ({ row }) => {
			const doctor = row.original;
			return (
				<DropdownMenu>
					<DropdownMenuTrigger
						render={
							<Button variant="ghost" className="h-8 w-8 p-0">
								<Text uuid="table.button.open_menu" className="sr-only" />
								<MoreHorizontal className="h-4 w-4" />
							</Button>
						}
					/>
					<DropdownMenuContent align="end">
						<DropdownMenuGroup>
							<DropdownMenuLabel>
								<Text uuid="table.actions" />
							</DropdownMenuLabel>
							<DropdownMenuItem onClick={() => onEdit(doctor)}>
								<Pencil className="mr-2 h-4 w-4" />
								<Text uuid="doctors.action.edit" />
							</DropdownMenuItem>
							<DropdownMenuItem onClick={() => onLink(doctor)}>
								<Link2 className="mr-2 h-4 w-4" />
								<Text uuid="doctors.action.link" />
							</DropdownMenuItem>
							<DropdownMenuItem onClick={() => onToggleActive(doctor)}>
								{doctor.active ? (
									<>
										<PowerOff className="mr-2 h-4 w-4" />
										<Text uuid="doctors.action.deactivate" />
									</>
								) : (
									<>
										<Power className="mr-2 h-4 w-4" />
										<Text uuid="doctors.action.activate" />
									</>
								)}
							</DropdownMenuItem>
							<DropdownMenuSeparator />
							<DropdownMenuItem
								onClick={() => onDelete(doctor)}
								className="text-red-600 focus:bg-red-100 focus:text-red-600 dark:focus:bg-red-900"
							>
								<Trash2 className="mr-2 h-4 w-4" />
								<Text uuid="doctors.action.delete" />
							</DropdownMenuItem>
						</DropdownMenuGroup>
					</DropdownMenuContent>
				</DropdownMenu>
			);
		},
	},
];
