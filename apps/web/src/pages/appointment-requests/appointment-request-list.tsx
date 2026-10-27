import { useText } from "@pengi/shared";
import { Text, ToggleGroup, ToggleGroupItem } from "@pengi/ui";
import React from "react";
import {
	type AppointmentRequest,
	type AppointmentRequestFilter,
	approveAppointmentRequest,
	getAppointmentRequests,
} from "@/api/appointment-request-service";
import { PageHeader } from "@/components/custom/page-header";
import { DataTable } from "@/components/custom/table/data-table";
import { RejectRequestDialog } from "@/components/features/appointment-requests/reject-request-dialog";
import { getAppointmentRequestColumns } from "@/sections/columns/appointment-requests/appointment-request-columns";
import { useAppointmentRequestsStore } from "@/store/appointment-requests-store";

const FILTERS: { value: AppointmentRequestFilter; labelKey: string }[] = [
	{ value: "pending", labelKey: "appointment_requests.filter.pending" },
	{ value: "all", labelKey: "appointment_requests.filter.all" },
];

/** `/appointment-requests`: what patients asked for from the portal, to approve or reject. */
export default function AppointmentRequestListPage() {
	const { textGet } = useText();
	const refreshPendingCount = useAppointmentRequestsStore(
		(s) => s.refreshPendingCount,
	);
	const [filter, setFilter] =
		React.useState<AppointmentRequestFilter>("pending");
	const [requests, setRequests] = React.useState<AppointmentRequest[]>([]);
	const [loading, setLoading] = React.useState(true);
	const [version, setVersion] = React.useState(0);
	const [busyId, setBusyId] = React.useState<number>();
	const [rejectTarget, setRejectTarget] =
		React.useState<AppointmentRequest | null>(null);

	React.useEffect(() => {
		let active = true;
		setLoading(true);
		getAppointmentRequests(filter).then((res) => {
			if (!active) return;
			setRequests(res.success ? (res.data ?? []) : []);
			setLoading(false);
		});
		return () => {
			active = false;
		};
	}, [filter, version]);

	const reload = React.useCallback(() => {
		setVersion((v) => v + 1);
		refreshPendingCount();
	}, [refreshPendingCount]);

	const handleApprove = React.useCallback(
		async (request: AppointmentRequest) => {
			setBusyId(request.id);
			await approveAppointmentRequest(request.id);
			setBusyId(undefined);
			// Approved, or taken meanwhile (the toast says why): either way reload.
			reload();
		},
		[reload],
	);

	const columns = React.useMemo(
		() =>
			getAppointmentRequestColumns({
				textGet,
				busyId,
				onApprove: handleApprove,
				onReject: setRejectTarget,
			}),
		[textGet, busyId, handleApprove],
	);

	return (
		<main className="grid grid-cols-1 items-start gap-4">
			<PageHeader
				title={textGet("appointment_requests.title")}
				description={textGet("appointment_requests.description")}
			/>
			<DataTable
				columns={columns}
				data={requests}
				loading={loading}
				toolbarRight={
					<ToggleGroup
						value={[filter]}
						onValueChange={(value) => {
							if (value[0]) setFilter(value[0] as AppointmentRequestFilter);
						}}
					>
						{FILTERS.map((f) => (
							<ToggleGroupItem key={f.value} value={f.value}>
								<Text uuid={f.labelKey} />
							</ToggleGroupItem>
						))}
					</ToggleGroup>
				}
				emptyState={
					<p className="py-8 text-center text-sm text-muted-foreground">
						{textGet(
							filter === "pending"
								? "appointment_requests.empty.pending"
								: "appointment_requests.empty.all",
						)}
					</p>
				}
			/>
			<RejectRequestDialog
				request={rejectTarget}
				onOpenChange={(open) => !open && setRejectTarget(null)}
				onRejected={reload}
			/>
		</main>
	);
}
