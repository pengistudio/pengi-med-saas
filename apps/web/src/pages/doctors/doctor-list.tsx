import { useText } from "@pengi/shared";
import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
	Button,
	Text,
} from "@pengi/ui";
import { CheckCheck, Plus } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import {
	activateDoctor,
	DOCTOR_IN_USE_ERROR,
	type Doctor,
	deactivateDoctor,
	deleteDoctor,
	markDoctorsReviewed,
} from "@/api/doctors-service";
import { PageHeader } from "@/components/custom/page-header";
import { DataTable } from "@/components/custom/table/data-table";
import { LinkUserDialog } from "@/components/features/doctors/link-user-dialog";
import { useTeamMembers } from "@/components/features/doctors/use-linkable-users";
import { getDoctorColumns } from "@/sections/columns/doctors/doctor-columns";
import {
	useDoctorStatus,
	useDoctorStore,
	useDoctors,
} from "@/store/doctors-store";

export default function DoctorListPage() {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { doctors, loaded } = useDoctors();
	const status = useDoctorStatus();
	const refresh = useDoctorStore((s) => s.refresh);
	const members = useTeamMembers();
	const [linkTarget, setLinkTarget] = React.useState<Doctor | null>(null);
	const [deleteTarget, setDeleteTarget] = React.useState<Doctor | null>(null);
	// Delete refused because documents reference it: offer to deactivate.
	const [inUseTarget, setInUseTarget] = React.useState<Doctor | null>(null);

	const userNames = React.useMemo(
		() =>
			new Map(
				members.map((m) => [m.user_id, m.environment_name || m.user_name]),
			),
		[members],
	);

	async function handleToggleActive(doctor: Doctor) {
		const res = doctor.active
			? await deactivateDoctor(doctor.ID)
			: await activateDoctor(doctor.ID);
		if (res.success) refresh();
	}

	async function handleDelete() {
		const doctor = deleteTarget;
		if (!doctor) return;
		setDeleteTarget(null);
		const res = await deleteDoctor(doctor.ID);
		if (res.success) {
			refresh();
		} else if (res.data?.error_code === DOCTOR_IN_USE_ERROR) {
			// Not toasted by the service: this dialog explains it instead.
			setInUseTarget(doctor);
		}
	}

	async function handleDeactivateInUse() {
		const doctor = inUseTarget;
		setInUseTarget(null);
		if (doctor) await handleToggleActive(doctor);
	}

	async function handleMarkReviewed() {
		const res = await markDoctorsReviewed();
		if (res.success) refresh();
	}

	const columns = getDoctorColumns({
		textGet,
		userNames,
		onEdit: (d) => navigate(`/doctors/edit/${d.ID}`),
		onLink: setLinkTarget,
		onToggleActive: handleToggleActive,
		onDelete: setDeleteTarget,
	});

	return (
		<main className="grid grid-cols-1 items-start gap-4">
			<PageHeader
				title={textGet("doctors.title")}
				description={textGet("doctors.page.description")}
				actions={
					<div className="flex flex-wrap gap-2">
						{status?.needs_review && (
							<Button variant="outline" onClick={handleMarkReviewed}>
								<CheckCheck className="mr-2 h-4 w-4" />
								<Text uuid="doctors.review.mark" />
							</Button>
						)}
						<Button onClick={() => navigate("/doctors/create")}>
							<Plus className="mr-2 h-4 w-4" />
							<Text uuid="doctors.create.button" />
						</Button>
					</div>
				}
			/>
			<DataTable
				columns={columns}
				data={doctors}
				loading={!loaded}
				searchKey="full_name"
				searchPlaceholder={textGet("doctors.search.placeholder")}
				emptyState={
					<p className="py-6 text-center text-sm text-muted-foreground">
						<Text uuid="doctors.empty" />
					</p>
				}
			/>

			<LinkUserDialog
				doctor={linkTarget}
				onOpenChange={(open) => !open && setLinkTarget(null)}
				onLinked={refresh}
			/>

			<AlertDialog
				open={deleteTarget !== null}
				onOpenChange={(open) => !open && setDeleteTarget(null)}
			>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>
							<Text uuid="dialog.title.absolutely.sure" />
						</AlertDialogTitle>
						<AlertDialogDescription>
							{textGet("doctors.delete.description", {
								name: deleteTarget?.full_name ?? "",
							})}
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>
							<Text uuid="form.cancel" />
						</AlertDialogCancel>
						<AlertDialogAction onClick={handleDelete}>
							<Text uuid="form.continue" />
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>

			<AlertDialog
				open={inUseTarget !== null}
				onOpenChange={(open) => !open && setInUseTarget(null)}
			>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>
							<Text uuid="doctors.in_use.title" />
						</AlertDialogTitle>
						<AlertDialogDescription>
							{textGet(
								inUseTarget?.active === false
									? "doctors.in_use.description_inactive"
									: "doctors.in_use.description",
								{ name: inUseTarget?.full_name ?? "" },
							)}
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>
							<Text uuid="form.cancel" />
						</AlertDialogCancel>
						{inUseTarget?.active !== false && (
							<AlertDialogAction onClick={handleDeactivateInUse}>
								<Text uuid="doctors.action.deactivate" />
							</AlertDialogAction>
						)}
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</main>
	);
}
