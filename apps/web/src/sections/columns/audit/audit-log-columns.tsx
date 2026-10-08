import { useText } from "@pengi/shared";
import { Badge, Button, DataTableColumnHeader, Text } from "@pengi/ui";
import type { ColumnDef } from "@tanstack/react-table";
import {
	AUDIT_ENTITY_TYPES,
	type AuditAction,
	type AuditLog,
} from "@/api/audit-service";
import { FormattedDate } from "@/components/custom/formatted";

const ACTION_VARIANT: Record<
	AuditAction,
	"secondary" | "default" | "outline" | "destructive"
> = {
	READ: "outline",
	CREATE: "default",
	UPDATE: "secondary",
	DELETE: "destructive",
};

/** Label of a record type; types without one show the table name. */
export function EntityTypeLabel({ type }: { type: string }) {
	const { textGet } = useText();
	const known = (AUDIT_ENTITY_TYPES as readonly string[]).includes(type);
	return <>{known ? textGet(`audit.entity.${type}`) : type}</>;
}

export function ActionBadge({ action }: { action: AuditAction }) {
	return (
		<Badge variant={ACTION_VARIANT[action] ?? "outline"}>
			<Text uuid={`audit.action.${action.toLowerCase()}`} />
		</Badge>
	);
}

export const getAuditLogColumns = (
	/** Opens the row's detail (values before and after). */
	onOpen: (log: AuditLog) => void,
): ColumnDef<AuditLog>[] => [
	{
		accessorKey: "created_at",
		meta: { title: "audit.column.date", phone: "end" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="audit.column.date" />}
			/>
		),
		cell: ({ row }) => (
			<FormattedDate
				value={row.original.created_at}
				withTime
				className="whitespace-nowrap tabular-nums"
			/>
		),
	},
	{
		accessorKey: "user_name",
		meta: { title: "audit.column.user", phone: "title" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="audit.column.user" />}
			/>
		),
		cell: ({ row }) => (
			<span className="font-medium">
				{row.original.user_name || `#${row.original.user_id}`}
			</span>
		),
	},
	{
		accessorKey: "action",
		meta: { title: "audit.column.action", phone: "status" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="audit.column.action" />}
			/>
		),
		cell: ({ row }) => <ActionBadge action={row.original.action} />,
	},
	{
		accessorKey: "entity_type",
		meta: { title: "audit.column.entity", phone: "subtitle" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="audit.column.entity" />}
			/>
		),
		cell: ({ row }) => (
			<span>
				<EntityTypeLabel type={row.original.entity_type} />{" "}
				<span className="text-muted-foreground tabular-nums">
					#{row.original.entity_id}
				</span>
			</span>
		),
	},
	{
		accessorKey: "patient_name",
		meta: { title: "audit.column.patient" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="audit.column.patient" />}
			/>
		),
		cell: ({ row }) => row.original.patient_name || "—",
	},
	{
		id: "actions",
		cell: ({ row }) => (
			<Button variant="ghost" size="sm" onClick={() => onOpen(row.original)}>
				<Text uuid="audit.detail.open" />
			</Button>
		),
	},
];
