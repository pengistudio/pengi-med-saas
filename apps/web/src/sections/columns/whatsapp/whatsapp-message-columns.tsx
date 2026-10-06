import { DataTableColumnHeader, Text } from "@pengi/ui";
import type { ColumnDef } from "@tanstack/react-table";
import {
	messageErrorKey,
	type WhatsAppMessage,
	type WhatsAppMessageStatus,
} from "@/api/whatsapp-service";
import { FormattedDate } from "@/components/custom/formatted";
import { cn } from "@/lib/utils";

const STATUS_STYLES: Record<WhatsAppMessageStatus, string> = {
	queued: "bg-muted text-muted-foreground",
	sending: "bg-muted text-muted-foreground",
	sent: "bg-blue-500/15 text-blue-700 dark:text-blue-400",
	delivered: "bg-blue-500/15 text-blue-700 dark:text-blue-400",
	read: "bg-teal-500/15 text-teal-700 dark:text-teal-400",
	replied: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-400",
	failed: "bg-destructive/10 text-destructive",
	skipped: "bg-amber-500/15 text-amber-700 dark:text-amber-400",
	received: "bg-muted text-muted-foreground",
};

function statusBadge(status: WhatsAppMessageStatus) {
	return (
		<span
			className={cn(
				"inline-flex w-fit items-center rounded-full px-2 py-0.5 text-xs font-medium",
				STATUS_STYLES[status] ?? STATUS_STYLES.queued,
			)}
		>
			<Text uuid={`settings.whatsapp.log.status.${status}`} />
		</span>
	);
}

/** Columns of the WhatsApp sent-messages log (inbox → "Envíos" tab). */
export const whatsappMessageColumns: ColumnDef<WhatsAppMessage>[] = [
	{
		accessorKey: "created_at",
		meta: { title: "settings.whatsapp.log.column.date", phone: "end" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="settings.whatsapp.log.column.date" />}
			/>
		),
		cell: ({ row }) => (
			<FormattedDate
				value={row.original.created_at}
				withTime
				className="whitespace-nowrap text-sm"
			/>
		),
	},
	{
		accessorKey: "patient_name",
		meta: { title: "settings.whatsapp.log.column.recipient", phone: "title" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="settings.whatsapp.log.column.recipient" />}
			/>
		),
		cell: ({ row }) => (
			<div className="grid">
				<span className="font-medium">
					{row.original.patient_name || (
						<Text uuid={`settings.whatsapp.log.kind.${row.original.kind}`} />
					)}
				</span>
				<span className="text-xs text-muted-foreground tabular-nums">
					{row.original.to_phone}
				</span>
			</div>
		),
	},
	{
		accessorKey: "offset_hours",
		meta: { title: "settings.whatsapp.log.column.reminder", phone: "subtitle" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="settings.whatsapp.log.column.reminder" />}
			/>
		),
		cell: ({ row }) =>
			row.original.kind === "reminder" ? (
				<Text
					uuid="settings.whatsapp.offset_before"
					values={{ count: row.original.offset_hours }}
				/>
			) : (
				<div className="grid max-w-xs">
					<Text uuid={`settings.whatsapp.log.kind.${row.original.kind}`} />
					{(row.original.kind === "reply" ||
						row.original.kind === "template" ||
						row.original.kind === "system") &&
						row.original.body && (
							<span className="truncate text-xs text-muted-foreground">
								{row.original.body}
							</span>
						)}
				</div>
			),
	},
	{
		accessorKey: "status",
		meta: { title: "settings.whatsapp.log.column.status", phone: "status" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="settings.whatsapp.log.column.status" />}
			/>
		),
		cell: ({ row }) => {
			const errorKey = messageErrorKey(row.original.error_code);
			return (
				<div className="grid gap-1">
					{statusBadge(row.original.status)}
					{errorKey && (
						<span
							className="max-w-xs text-xs whitespace-normal text-muted-foreground"
							title={row.original.error_detail || undefined}
						>
							<Text uuid={errorKey} />
						</span>
					)}
				</div>
			);
		},
	},
	{
		accessorKey: "reply",
		meta: { title: "settings.whatsapp.log.column.reply" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="settings.whatsapp.log.column.reply" />}
			/>
		),
		cell: ({ row }) =>
			row.original.reply ? (
				<span
					className={cn(
						"text-sm font-medium",
						row.original.reply === "confirm"
							? "text-emerald-700 dark:text-emerald-400"
							: "text-orange-600 dark:text-orange-400",
					)}
				>
					<Text uuid={`settings.whatsapp.log.reply.${row.original.reply}`} />
				</span>
			) : (
				<span className="text-muted-foreground">—</span>
			),
	},
];
