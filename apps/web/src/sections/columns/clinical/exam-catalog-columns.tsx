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
import type { ExamCatalogItem, ExamProfile } from "@/api/exam-order-service";
import { Switch } from "@/components/ui/switch";
import { EXAM_CATEGORY_KEYS } from "@/lib/exam-orders";

/** Exams shown by name in a profile row; the rest count as "+K más". */
const PROFILE_NAMES_SHOWN = 3;

function ProfileExamsCell({ profile }: { profile: ExamProfile }) {
	const { textGet } = useText();
	const names = (profile.items ?? []).map((item) => item.name);
	const rest = names.length - PROFILE_NAMES_SHOWN;
	return (
		<span
			className="block max-w-xs truncate text-sm text-muted-foreground"
			title={names.join(", ")}
		>
			{names.slice(0, PROFILE_NAMES_SHOWN).join(", ")}
			{rest > 0 &&
				` ${textGet("clinical.exam_catalog.profile.more", { count: rest })}`}
		</span>
	);
}

interface RowActions<T> {
	onEdit: (row: T) => void;
	onDelete: (row: T) => void;
	onToggleActive: (row: T, active: boolean) => void;
}

function actionsColumn<T extends { ID: number }>({
	onEdit,
	onDelete,
}: Pick<RowActions<T>, "onEdit" | "onDelete">): ColumnDef<T> {
	return {
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
						<DropdownMenuItem onClick={() => onEdit(row.original)}>
							<Pencil className="mr-2 h-4 w-4" />
							<Text uuid="clinical.exam_catalog.action.edit" />
						</DropdownMenuItem>
						<DropdownMenuItem
							className="text-destructive"
							onClick={() => onDelete(row.original)}
						>
							<Trash2 className="mr-2 h-4 w-4" />
							<Text uuid="clinical.exam_catalog.action.delete" />
						</DropdownMenuItem>
					</DropdownMenuGroup>
				</DropdownMenuContent>
			</DropdownMenu>
		),
	};
}

function activeColumn<T extends { ID: number; active: boolean }>(
	onToggleActive: RowActions<T>["onToggleActive"],
): ColumnDef<T> {
	return {
		accessorKey: "active",
		meta: { title: "clinical.exam_catalog.column.active", phone: "end" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="clinical.exam_catalog.column.active" />}
			/>
		),
		cell: ({ row }) => (
			<Switch
				checked={row.original.active}
				onCheckedChange={(checked) => onToggleActive(row.original, checked)}
			/>
		),
	};
}

export const getExamCatalogColumns = (
	actions: RowActions<ExamCatalogItem>,
): ColumnDef<ExamCatalogItem>[] => [
	{
		accessorKey: "name",
		meta: { title: "clinical.exam_catalog.field.name", phone: "title" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="clinical.exam_catalog.field.name" />}
			/>
		),
		cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
	},
	{
		accessorKey: "category",
		meta: { title: "clinical.exam_orders.field.category", phone: "subtitle" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="clinical.exam_orders.field.category" />}
			/>
		),
		cell: ({ row }) => (
			<Text uuid={EXAM_CATEGORY_KEYS[row.original.category]} />
		),
	},
	{
		accessorKey: "subgroup",
		meta: { title: "clinical.exam_catalog.field.subgroup", phone: "status" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="clinical.exam_catalog.field.subgroup" />}
			/>
		),
		cell: ({ row }) => <span>{row.original.subgroup || "—"}</span>,
	},
	{
		accessorKey: "default_indications",
		meta: { title: "clinical.exam_catalog.field.default_indications" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="clinical.exam_catalog.field.default_indications" />}
			/>
		),
		cell: ({ row }) => (
			<span
				className="line-clamp-2 block max-w-xs text-sm whitespace-normal text-muted-foreground"
				title={row.original.default_indications}
			>
				{row.original.default_indications || "—"}
			</span>
		),
	},
	activeColumn<ExamCatalogItem>(actions.onToggleActive),
	actionsColumn<ExamCatalogItem>(actions),
];

export const getExamProfileColumns = (
	actions: RowActions<ExamProfile>,
): ColumnDef<ExamProfile>[] => [
	{
		accessorKey: "name",
		meta: { title: "clinical.exam_catalog.field.name", phone: "title" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="clinical.exam_catalog.field.name" />}
			/>
		),
		cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
	},
	{
		id: "exams",
		meta: { title: "clinical.exam_orders.column.exams", phone: "subtitle" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="clinical.exam_orders.column.exams" />}
			/>
		),
		cell: ({ row }) => <ProfileExamsCell profile={row.original} />,
	},
	activeColumn<ExamProfile>(actions.onToggleActive),
	actionsColumn<ExamProfile>(actions),
];
