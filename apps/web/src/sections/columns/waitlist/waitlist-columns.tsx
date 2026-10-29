import type { AppText } from "@pengi/shared";
import type { DataTableColumnDef } from "@pengi/ui";
import { Badge, Button, Text } from "@pengi/ui";
import { Pencil, Trash2 } from "lucide-react";
import type React from "react";
import {
	type OfferChannel,
	type WaitlistEntry,
	type WaitlistStatus,
	waitlistPatientName,
} from "@/api/waitlist-service";
import { FormattedDate } from "@/components/custom/formatted";
import { RelativeDate } from "@/components/custom/relative-date";
import {
	entryPreference,
	WAITLIST_STATUS_VARIANT,
} from "@/components/features/waitlist/waitlist-utils";

// Literal keys (checked by the message-keys test).
const STATUS_TEXT: Record<WaitlistStatus, React.ReactNode> = {
	waiting: <Text uuid="waitlist.status.waiting" />,
	offered: <Text uuid="waitlist.status.offered" />,
	booked: <Text uuid="waitlist.status.booked" />,
	removed: <Text uuid="waitlist.status.removed" />,
	expired: <Text uuid="waitlist.status.expired" />,
};

const CHANNEL_TEXT: Record<OfferChannel, React.ReactNode> = {
	whatsapp: <Text uuid="waitlist.offer.channel.whatsapp" />,
	phone: <Text uuid="waitlist.offer.channel.phone" />,
};

const SOURCE_TEXT: Record<WaitlistEntry["source"], React.ReactNode> = {
	staff: <Text uuid="waitlist.source.staff" />,
	portal: <Text uuid="waitlist.source.portal" />,
};

interface WaitlistColumnProps {
	textGet: AppText["textGet"];
	formatDate: AppText["formatDate"];
	/** MANAGE_APPOINTMENT: edit and remove show. */
	canManage: boolean;
	onEdit: (entry: WaitlistEntry) => void;
	onRemove: (entry: WaitlistEntry) => void;
}

export const getWaitlistColumns = ({
	textGet,
	formatDate,
	canManage,
	onEdit,
	onRemove,
}: WaitlistColumnProps): DataTableColumnDef<WaitlistEntry>[] => [
	{
		id: "patient",
		accessorFn: waitlistPatientName,
		meta: { title: "waitlist.column.patient", phone: "title" },
		header: () => <Text uuid="waitlist.column.patient" />,
		cell: ({ row }) => (
			<div className="min-w-0">
				<p className="font-medium">{waitlistPatientName(row.original)}</p>
				<p className="text-sm text-muted-foreground tabular-nums">
					{row.original.patient.phone || row.original.patient.document}
				</p>
			</div>
		),
	},
	{
		id: "doctor",
		accessorFn: (e) => e.doctor?.full_name ?? "",
		meta: { title: "waitlist.column.doctor", phone: "subtitle" },
		header: () => <Text uuid="waitlist.column.doctor" />,
		cell: ({ row }) =>
			row.original.doctor ? (
				row.original.doctor.full_name
			) : (
				<span className="text-muted-foreground">
					<Text uuid="waitlist.doctor.any" />
				</span>
			),
	},
	{
		id: "preference",
		accessorFn: (e) => entryPreference(e, textGet, formatDate),
		meta: { title: "waitlist.column.preference", phone: "detail" },
		header: () => <Text uuid="waitlist.column.preference" />,
		cell: ({ row }) => (
			<div className="max-w-72">
				<p className="text-sm">
					{entryPreference(row.original, textGet, formatDate)}
				</p>
				{row.original.notes && (
					<p className="text-sm whitespace-pre-line text-muted-foreground">
						{row.original.notes}
					</p>
				)}
			</div>
		),
	},
	{
		accessorKey: "status",
		meta: { title: "waitlist.column.status", phone: "end" },
		header: () => <Text uuid="waitlist.column.status" />,
		cell: ({ row }) => {
			const { status, offer } = row.original;
			return (
				<div className="flex flex-col items-start gap-1">
					<Badge variant={WAITLIST_STATUS_VARIANT[status] ?? "outline"}>
						{STATUS_TEXT[status]}
					</Badge>
					{status === "offered" && offer && (
						<span className="text-xs text-muted-foreground">
							<FormattedDate value={offer.date} style="day-month" />{" "}
							<span className="tabular-nums">{offer.start_time}</span>
							{" · "}
							{CHANNEL_TEXT[offer.channel]}
							{offer.accepted_at && (
								<>
									{" · "}
									<Text uuid="waitlist.offer.accepted" />
								</>
							)}
						</span>
					)}
				</div>
			);
		},
	},
	{
		id: "added",
		accessorFn: (e) => e.created_at,
		meta: { title: "waitlist.column.added", phone: "detail" },
		header: () => <Text uuid="waitlist.column.added" />,
		cell: ({ row }) => (
			<div className="flex flex-col text-sm">
				<RelativeDate
					date={row.original.created_at}
					className="whitespace-nowrap"
				/>
				<span className="text-xs text-muted-foreground">
					{SOURCE_TEXT[row.original.source]}
				</span>
			</div>
		),
	},
	{
		id: "actions",
		cell: ({ row }) => {
			const entry = row.original;
			const active = entry.status === "waiting" || entry.status === "offered";
			if (!canManage || !active) return null;
			return (
				<div className="flex justify-end gap-1">
					<Button
						size="icon"
						variant="ghost"
						aria-label={textGet("waitlist.action.edit")}
						onClick={() => onEdit(entry)}
					>
						<Pencil className="h-4 w-4" />
					</Button>
					<Button
						size="icon"
						variant="ghost"
						aria-label={textGet("waitlist.action.remove")}
						onClick={() => onRemove(entry)}
					>
						<Trash2 className="h-4 w-4" />
					</Button>
				</div>
			);
		},
	},
];
