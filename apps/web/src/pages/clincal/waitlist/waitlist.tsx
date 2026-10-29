import { useText } from "@pengi/shared";
import {
	Button,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
	Text,
	ToggleGroup,
	ToggleGroupItem,
} from "@pengi/ui";
import { Plus } from "lucide-react";
import React from "react";
import { useSearchParams } from "react-router";
import {
	createWaitlistEntry,
	getWaitlist,
	removeWaitlistEntry,
	slotFromSearch,
	updateWaitlistEntry,
	type WaitlistEntry,
	type WaitlistEntryPayload,
	type WaitlistStatusFilter,
} from "@/api/waitlist-service";
import { PageHeader } from "@/components/custom/page-header";
import { DataTable } from "@/components/custom/table/data-table";
import { FreedSlotPanel } from "@/components/features/waitlist/freed-slot-panel";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS } from "@/lib/constants";
import { getWaitlistColumns } from "@/sections/columns/waitlist/waitlist-columns";
import { WaitlistEntryForm } from "@/sections/forms/waitlist/waitlist-entry-form";
import { useDoctors } from "@/store/doctors-store";

const ALL_DOCTORS = "all";
const ANY_DOCTOR = "any";

/**
 * `/clinical/waitlist`: patients waiting for a slot. A freed-slot link
 * (`?date&start_time&end_time&doctor_id&appointment_type_id`, from the
 * notification) shows that slot's candidates on top, to offer it.
 */
export default function WaitlistPage() {
	const { textGet, formatDate } = useText();
	const { checkPermission } = usePermission();
	const canManage = checkPermission([
		PERMISSIONS.APPOINTMENTS.PERMISSION_MANAGE_APPOINTMENT,
	]);
	const { activeDoctors } = useDoctors();
	const [searchParams, setSearchParams] = useSearchParams();
	const searchKey = searchParams.toString();
	const slot = React.useMemo(
		() => slotFromSearch(new URLSearchParams(searchKey)),
		[searchKey],
	);

	const [status, setStatus] = React.useState<WaitlistStatusFilter>("active");
	const [doctor, setDoctor] = React.useState(ALL_DOCTORS);
	const [entries, setEntries] = React.useState<WaitlistEntry[]>([]);
	const [loading, setLoading] = React.useState(true);
	const [version, setVersion] = React.useState(0);
	const [editing, setEditing] = React.useState<WaitlistEntry | "new" | null>(
		null,
	);
	const [saving, setSaving] = React.useState(false);

	React.useEffect(() => {
		let active = true;
		setLoading(true);
		getWaitlist({
			status,
			doctor:
				doctor === ALL_DOCTORS
					? undefined
					: doctor === ANY_DOCTOR
						? "any"
						: Number(doctor),
		}).then((res) => {
			if (!active) return;
			setEntries(res.success ? (res.data ?? []) : []);
			setLoading(false);
		});
		return () => {
			active = false;
		};
	}, [status, doctor, version]);

	const reload = React.useCallback(() => setVersion((v) => v + 1), []);

	const handleRemove = React.useCallback(
		async (entry: WaitlistEntry) => {
			const res = await removeWaitlistEntry(entry.id);
			if (res.success) reload();
		},
		[reload],
	);

	const save = async (payload: WaitlistEntryPayload) => {
		setSaving(true);
		const res =
			editing && editing !== "new"
				? await updateWaitlistEntry(editing.id, payload)
				: await createWaitlistEntry(payload);
		setSaving(false);
		if (res.success) {
			setEditing(null);
			reload();
		}
	};

	const columns = React.useMemo(
		() =>
			getWaitlistColumns({
				textGet,
				formatDate,
				canManage,
				onEdit: setEditing,
				onRemove: handleRemove,
			}),
		[textGet, formatDate, canManage, handleRemove],
	);

	return (
		<main className="grid grid-cols-1 items-start gap-4">
			<PageHeader
				title={textGet("waitlist.title")}
				description={textGet("waitlist.description")}
				actions={
					canManage && (
						<Button onClick={() => setEditing("new")}>
							<Plus className="h-4 w-4" />
							<Text uuid="waitlist.action.add" />
						</Button>
					)
				}
			/>
			{slot && (
				<FreedSlotPanel
					slot={slot}
					canManage={canManage}
					onClose={() => setSearchParams({})}
					onOffered={reload}
				/>
			)}
			<DataTable
				columns={columns}
				data={entries}
				loading={loading}
				toolbarRight={
					<div className="flex flex-wrap items-center gap-2">
						<Select
							value={doctor}
							onValueChange={(value) => setDoctor(String(value ?? ALL_DOCTORS))}
						>
							<SelectTrigger
								className="w-56"
								aria-label={textGet("waitlist.filter.doctor")}
							>
								<SelectValue />
							</SelectTrigger>
							<SelectContent>
								<SelectItem value={ALL_DOCTORS}>
									{textGet("waitlist.filter.doctor.all")}
								</SelectItem>
								<SelectItem value={ANY_DOCTOR}>
									{textGet("waitlist.doctor.any")}
								</SelectItem>
								{activeDoctors.map((d) => (
									<SelectItem key={d.ID} value={String(d.ID)}>
										{d.full_name}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
						<ToggleGroup
							value={[status]}
							onValueChange={(value) => {
								if (value[0]) setStatus(value[0] as WaitlistStatusFilter);
							}}
						>
							<ToggleGroupItem value="active">
								<Text uuid="waitlist.filter.active" />
							</ToggleGroupItem>
							<ToggleGroupItem value="all">
								<Text uuid="waitlist.filter.all" />
							</ToggleGroupItem>
						</ToggleGroup>
					</div>
				}
				emptyState={
					<p className="py-8 text-center text-sm text-muted-foreground">
						{status === "active"
							? textGet("waitlist.empty.active")
							: textGet("waitlist.empty.all")}
					</p>
				}
			/>
			<Dialog
				open={editing !== null}
				onOpenChange={(open) => !open && setEditing(null)}
			>
				<DialogContent className="sm:max-w-[560px]">
					<DialogHeader>
						<DialogTitle>
							{editing === "new"
								? textGet("waitlist.form.title.new")
								: textGet("waitlist.form.title.edit")}
						</DialogTitle>
						<DialogDescription>
							{textGet("waitlist.form.description")}
						</DialogDescription>
					</DialogHeader>
					{editing !== null && (
						<WaitlistEntryForm
							key={editing === "new" ? "new" : editing.id}
							entry={editing === "new" ? undefined : editing}
							loading={saving}
							onSubmit={save}
							onCancel={() => setEditing(null)}
						/>
					)}
				</DialogContent>
			</Dialog>
		</main>
	);
}
