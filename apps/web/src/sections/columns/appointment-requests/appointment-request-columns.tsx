import type { DataTableColumnDef } from "@pengi/ui";
import { Badge, Button, Text } from "@pengi/ui";
import { Check, X } from "lucide-react";
import {
	type AppointmentRequest,
	type AppointmentRequestStatus,
	requesterName,
} from "@/api/appointment-request-service";
import { FormattedDate } from "@/components/custom/formatted";
import { RelativeDate } from "@/components/custom/relative-date";
import { RequesterCell } from "@/components/features/appointment-requests/requester-cell";
import { specialtyLabel } from "@/components/features/doctors/doctor-utils";

interface AppointmentRequestColumnProps {
	textGet: (key: string) => string;
	/** Request being approved or rejected: its buttons wait. */
	busyId?: number;
	onApprove: (request: AppointmentRequest) => void;
	onReject: (request: AppointmentRequest) => void;
}

const STATUS_VARIANT: Record<
	AppointmentRequestStatus,
	"default" | "secondary" | "outline" | "destructive"
> = {
	pending: "default",
	approved: "secondary",
	rejected: "destructive",
	cancelled: "outline",
	expired: "outline",
};

const hhmm = (time: string) => time.slice(0, 5);

export const getAppointmentRequestColumns = ({
	textGet,
	busyId,
	onApprove,
	onReject,
}: AppointmentRequestColumnProps): DataTableColumnDef<AppointmentRequest>[] => [
	{
		id: "patient",
		accessorFn: requesterName,
		meta: { title: "appointment_requests.column.patient", phone: "title" },
		header: () => <Text uuid="appointment_requests.column.patient" />,
		cell: ({ row }) => <RequesterCell request={row.original} />,
	},
	{
		id: "when",
		accessorFn: (r) => `${r.date} ${r.start_time}`,
		meta: { title: "appointment_requests.column.when", phone: "subtitle" },
		header: () => <Text uuid="appointment_requests.column.when" />,
		cell: ({ row }) => (
			<span className="whitespace-nowrap">
				<FormattedDate value={row.original.date} style="weekday-day-month" />{" "}
				<span className="tabular-nums">
					{hhmm(row.original.start_time)}–{hhmm(row.original.end_time)}
				</span>
			</span>
		),
	},
	{
		id: "doctor",
		accessorFn: (r) => r.doctor.full_name,
		meta: { title: "appointment_requests.column.doctor", phone: "detail" },
		header: () => <Text uuid="appointment_requests.column.doctor" />,
		cell: ({ row }) => (
			<div className="min-w-0">
				<p>{row.original.doctor.full_name}</p>
				<p className="text-sm text-muted-foreground">
					{[
						row.original.type_name,
						row.original.doctor.specialty &&
							specialtyLabel(row.original.doctor, textGet),
					]
						.filter(Boolean)
						.join(" · ")}
				</p>
			</div>
		),
	},
	{
		accessorKey: "reason",
		meta: { title: "appointment_requests.column.reason", phone: "detail" },
		header: () => <Text uuid="appointment_requests.column.reason" />,
		cell: ({ row }) => {
			const { reason, reject_reason, status } = row.original;
			const text =
				status === "rejected" && reject_reason ? reject_reason : reason;
			return text ? (
				<p className="max-w-64 text-sm whitespace-pre-line">{text}</p>
			) : (
				<span className="text-muted-foreground">—</span>
			);
		},
	},
	{
		accessorKey: "status",
		meta: { title: "appointment_requests.column.status", phone: "end" },
		header: () => <Text uuid="appointment_requests.column.status" />,
		cell: ({ row }) => {
			const { status, expires_at } = row.original;
			return (
				<div className="flex flex-col items-start gap-1">
					<Badge variant={STATUS_VARIANT[status] ?? "outline"}>
						<Text uuid={`appointment_requests.status.${status}`} />
					</Badge>
					{status === "pending" && expires_at && (
						<span className="text-xs text-muted-foreground">
							<Text uuid="appointment_requests.expires" />{" "}
							<RelativeDate date={expires_at} className="whitespace-nowrap" />
						</span>
					)}
				</div>
			);
		},
	},
	{
		id: "actions",
		cell: ({ row }) => {
			const request = row.original;
			if (request.status !== "pending") return null;
			const busy = busyId === request.id;
			return (
				<div className="flex justify-end gap-2">
					<Button
						size="sm"
						variant="outline"
						disabled={busy}
						onClick={() => onReject(request)}
					>
						<X className="h-4 w-4" />
						<Text uuid="appointment_requests.action.reject" />
					</Button>
					<Button size="sm" disabled={busy} onClick={() => onApprove(request)}>
						<Check className="h-4 w-4" />
						<Text uuid="appointment_requests.action.approve" />
					</Button>
				</div>
			);
		},
	},
];
