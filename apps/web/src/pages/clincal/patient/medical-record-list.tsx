import { useText } from "@pengi/shared";
import {
	Badge,
	Button,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
	Text,
	useToast,
} from "@pengi/ui";
import type { Row } from "@tanstack/react-table";
import {
	ArchiveRestore,
	Cake,
	ChevronRight,
	CircleAlert,
	Files,
	IdCard,
	MessageCircle,
	MoreHorizontal,
	Pencil,
	Pill,
	Plus,
	Trash,
	TriangleAlert,
} from "lucide-react";
import React from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";
import {
	bulkDeleteMedicalRecords,
	type DeletedMedicalRecord,
	deleteMedicalRecord,
	downloadPrescription,
	getDeletedMedicalRecords,
	getMedicalRecords,
	getPatientById,
	type MedicalRecord,
	type Patient,
	restoreMedicalRecord,
	updateCritical,
	updateCriticalRevert,
} from "@/api/clinical-service";
import { PageHeader } from "@/components/custom/page-header";
import PatientCard, { formatAge } from "@/components/custom/patient-card";
import { DataTable } from "@/components/custom/table/data-table";
import EditPrescriptionDialog from "@/components/features/patient/edit-prescription-dialog";
import PrescriptionDialog from "@/components/features/patient/prescription-dialog";
import { OpenChatButton } from "@/components/features/whatsapp/open-chat-button";
import usePermission from "@/hooks/use-permission";
import { parseAllergies } from "@/lib/allergies";
import { EMPTY_STRING, PERMISSIONS } from "@/lib/constants";
import { ageInYears } from "@/lib/patient-age";
import {
	buildPrescriptionWhatsAppMessage,
	generateWhatsAppLink,
} from "@/lib/utils";
import { getDeletedMedicalRecordColumns } from "@/sections/columns/clinical/deleted-medical-record-columns";
import { getMedicalRecordColumns } from "@/sections/columns/clinical/medical-record-columns";
import MedicalRecordDeleteForm, {
	type MedicalRecordDeleteValues,
} from "@/sections/forms/clinical/medical-record-delete-form";
import {
	selectPatient,
	selectSetPatient,
	usePatientStore,
} from "@/store/patient-store";
import { useRowStore } from "@/store/row-store";

const MedicalRecords = () => {
	const patient = usePatientStore(selectPatient);
	const setPatient = usePatientStore(selectSetPatient);
	const { patientId } = useParams<{ patientId: string }>();
	const navigate = useNavigate();
	// Set when coming from the waiting room: the new consultation is linked to
	// that appointment (and gets its triage vital signs).
	const [searchParams] = useSearchParams();
	const appointmentId = searchParams.get("appointment_id");

	React.useEffect(() => {
		if (!patientId) return;
		getPatientById(Number(patientId)).then((res) => {
			if (res.success && res.data) setPatient(res.data as Patient);
		});
	}, [patientId, setPatient]);
	const { infoToast } = useToast();
	const { checkPermission } = usePermission();
	const { textGet, formatDate } = useText();

	const [medicalRecords, setMedicalRecords] = React.useState<MedicalRecord[]>(
		[],
	);
	const [page, setPage] = React.useState(1);
	const [totalPages, setTotalPages] = React.useState(1);
	const [loading, setLoading] = React.useState(true);
	const [showPrescription, setShowPrescription] = React.useState(false);
	const [showEditPrescription, setShowEditPrescription] = React.useState(false);
	const [selectedRecord, setSelectedRecord] =
		React.useState<MedicalRecord | null>(null);

	// Soft delete with a reason; the deleted ones are listed apart and can be
	// restored (DELETE_MEDICAL_RECORD).
	const canDelete = checkPermission([
		PERMISSIONS.MEDICAL_RECORD.PERMISSION_DELETE_MEDICAL_RECORD,
	]);
	// The records being deleted: one from its row menu, or the selected rows.
	const [deleting, setDeleting] = React.useState<MedicalRecord[]>([]);
	const { rows: selectedRows } = useRowStore();
	const [deletingBusy, setDeletingBusy] = React.useState(false);
	const [showDeleted, setShowDeleted] = React.useState(false);
	const [deletedRecords, setDeletedRecords] = React.useState<
		DeletedMedicalRecord[]
	>([]);
	const [deletedLoading, setDeletedLoading] = React.useState(false);
	const [restoringId, setRestoringId] = React.useState<number | null>(null);

	const loadDeleted = React.useCallback(() => {
		if (!patient) return;
		getDeletedMedicalRecords(patient.ID).then((res) => {
			setDeletedRecords(res.success ? (res.data ?? []) : []);
			setDeletedLoading(false);
		});
	}, [patient]);

	React.useEffect(() => {
		if (showDeleted) loadDeleted();
	}, [showDeleted, loadDeleted]);

	// Show the spinner as soon as the query changes (adjust state during render).
	const [prevQuery, setPrevQuery] = React.useState({ patient, page });
	if (prevQuery.patient !== patient || prevQuery.page !== page) {
		setPrevQuery({ patient, page });
		if (patient) setLoading(true);
	}

	React.useEffect(() => {
		if (!patient) return;
		getMedicalRecords(patient.ID, { page, limit: 10 })
			.then((res) => {
				if (res.success && res.data) {
					setMedicalRecords(res.data.items);
					setTotalPages(res.data.total_pages);
				}
			})
			.finally(() => {
				setLoading(false);
			});
	}, [patient, page]);

	const latestPrescription = React.useMemo(() => {
		const withPrescription = medicalRecords.filter((r) => r.prescription);
		if (withPrescription.length === 0) return null;

		const sorted = [...withPrescription].sort(
			(a, b) => new Date(b.date).getTime() - new Date(a.date).getTime(),
		);
		return sorted[0].prescription;
	}, [medicalRecords]);

	const displayName = patient
		? patient.full_name || `${patient.first_name} ${patient.last_name}`.trim()
		: "";
	const age = ageInYears(patient?.birth_date);
	const allergies = parseAllergies(patient?.allergies);

	return (
		<main className="grid flex-1 items-start gap-6 auto-rows-max">
			{patient && (
				<div className="space-y-3">
					<nav className="flex items-center gap-1.5 text-sm text-muted-foreground">
						<button
							type="button"
							onClick={() => navigate("/clinical")}
							className="rounded-sm transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
						>
							<Text uuid="dashboard.clinical.patients" />
						</button>
						<ChevronRight className="h-3.5 w-3.5" />
						<span className="truncate text-foreground">{displayName}</span>
					</nav>
					<PageHeader
						title={
							<span className="flex flex-wrap items-center gap-x-3 gap-y-2">
								{displayName}
								{patient.critical && (
									<Badge variant="destructive" className="font-medium">
										<CircleAlert />
										<Text uuid="clinical.patient_card.critical" />
									</Badge>
								)}
								{allergies.length > 0 && (
									<Badge
										variant="outline"
										className="border-amber-300 bg-amber-50 font-medium text-amber-800 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-300"
										title={allergies.join(", ")}
									>
										<TriangleAlert />
										<Text uuid="clinical.patient_card.has_allergies" />
									</Badge>
								)}
							</span>
						}
						description={
							<span className="flex flex-wrap items-center gap-x-5 gap-y-1">
								<span className="inline-flex items-center gap-1.5">
									<IdCard className="h-4 w-4" />
									<span className="tabular-nums text-foreground">
										{patient.document}
									</span>
								</span>
								{age !== null && (
									<span className="inline-flex items-center gap-1.5">
										<Cake className="h-4 w-4" />
										{formatAge(age, patient.birth_date_estimated, textGet)}
									</span>
								)}
							</span>
						}
						actions={
							<>
								<Button variant="outline" onClick={handleContactWhatsApp}>
									<MessageCircle />
									<Text uuid="clinical.medical_record.contact_ws" />
								</Button>
								<OpenChatButton patient={patient} />
								{checkPermission([
									PERMISSIONS.MEDICAL_RECORD.PERMISSION_UPDATE_PATIENT,
								]) && (
									<Button variant="outline" onClick={handleEditPatient}>
										<Pencil />
										<Text uuid="clinical.medical_record.edit_patient" />
									</Button>
								)}
								<DropdownMenu>
									<DropdownMenuTrigger
										render={
											<Button
												variant="outline"
												size="icon"
												aria-label={textGet("clinical.medical_record.actions")}
												title={textGet("clinical.medical_record.actions")}
											>
												<MoreHorizontal />
											</Button>
										}
									/>
									<DropdownMenuContent align="end" className="w-56">
										<DropdownMenuItem
											disabled={!latestPrescription}
											onClick={() => {
												setSelectedRecord(null);
												setShowPrescription(true);
											}}
										>
											<Pill />
											<Text uuid="clinical.medical_record.view_prescriptions" />
										</DropdownMenuItem>
										<DropdownMenuSeparator />
										{patient.critical ? (
											<DropdownMenuItem onClick={handleCriticalRevert}>
												<CircleAlert />
												<Text uuid="clinical.medical_record.unmark_critical" />
											</DropdownMenuItem>
										) : (
											<DropdownMenuItem
												variant="destructive"
												onClick={handleCritical}
											>
												<TriangleAlert />
												<Text uuid="clinical.medical_record.mark_critical" />
											</DropdownMenuItem>
										)}
									</DropdownMenuContent>
								</DropdownMenu>
								{checkPermission([
									PERMISSIONS.MEDICAL_RECORD.PERMISSION_CREATE_MEDICAL_RECORD,
								]) && (
									<Button onClick={handleCreateMedicalRecord}>
										<Plus />
										<Text uuid="clinical.medical_record.new_consultation" />
									</Button>
								)}
							</>
						}
					/>
				</div>
			)}
			<div className="grid grid-cols-1 items-start gap-6 lg:grid-cols-[340px_minmax(0,1fr)]">
				{patient && (
					<PatientCard patient={patient} onEditPatient={handleEditPatient} />
				)}
				<section className="min-w-0">
					<div className="flex items-center justify-between gap-2">
						<h2 className="text-lg font-semibold">
							<Text
								uuid={
									showDeleted
										? "clinical.medical_record.deleted.title"
										: "clinical.medical_record.history_title"
								}
							/>
						</h2>
						{canDelete && (
							<Button
								type="button"
								size="sm"
								variant="ghost"
								className="text-muted-foreground"
								onClick={() => {
									setDeletedLoading(!showDeleted);
									setShowDeleted((v) => !v);
								}}
							>
								{showDeleted ? <Files /> : <ArchiveRestore />}
								<Text
									uuid={
										showDeleted
											? "clinical.medical_record.deleted.hide"
											: "clinical.medical_record.deleted.show"
									}
								/>
							</Button>
						)}
					</div>
					{showDeleted ? (
						<DataTable
							columns={getDeletedMedicalRecordColumns(
								handleRestore,
								restoringId,
							)}
							data={deletedRecords}
							loading={deletedLoading}
							emptyState={
								<p className="py-4 text-sm text-muted-foreground">
									{textGet("clinical.medical_record.deleted.empty")}
								</p>
							}
						/>
					) : (
						<DataTable
							searchKey="motive"
							searchPlaceholder={textGet(
								"clinical.medical_record.search.placeholder",
							)}
							columns={getMedicalRecordColumns(
								handleView,
								handleEdit,
								handleViewPrescription,
								handleEditPrescription,
								handleDownloadPrescription,
								handleSendWhatsAppPrescription,
								canDelete ? (record) => setDeleting([record]) : undefined,
							)}
							bulkActions={
								canDelete ? (
									<Button
										variant="outline"
										onClick={() =>
											setDeleting(
												(selectedRows as Row<MedicalRecord>[]).map(
													(row) => row.original,
												),
											)
										}
									>
										<Trash />
										<Text uuid="table.button.delete.all.selected" />
									</Button>
								) : undefined
							}
							data={medicalRecords}
							loading={loading}
							pageCount={totalPages}
							page={page}
							onPageChange={setPage}
							emptyState={
								<div className="flex flex-col items-center gap-3 py-4">
									<p className="text-sm text-muted-foreground">
										{textGet("clinical.medical_record.empty")}
									</p>
									{checkPermission([
										PERMISSIONS.MEDICAL_RECORD.PERMISSION_CREATE_MEDICAL_RECORD,
									]) && (
										<button
											type="button"
											onClick={handleCreateMedicalRecord}
											className="text-sm text-primary underline-offset-4 hover:underline"
										>
											{textGet("clinical.medical_record.new_consultation")}
										</button>
									)}
								</div>
							}
						/>
					)}
				</section>
			</div>

			<PrescriptionDialog
				open={showPrescription}
				onOpenChange={setShowPrescription}
				prescription={selectedRecord?.prescription || latestPrescription}
			/>

			<Dialog
				open={deleting.length > 0}
				onOpenChange={(open) => {
					if (!open) setDeleting([]);
				}}
			>
				<DialogContent className="sm:max-w-[525px]">
					<DialogHeader>
						<DialogTitle>
							{textGet(
								deleting.length > 1
									? "clinical.medical_record.delete.bulk.title"
									: "clinical.medical_record.delete.title",
							)}
						</DialogTitle>
						<DialogDescription>
							{deleting.length > 1
								? textGet("clinical.medical_record.delete.bulk.description", {
										count: deleting.length,
									})
								: deleting.length === 1 &&
									textGet("clinical.medical_record.delete.description", {
										date: formatDate(deleting[0].date),
									})}
						</DialogDescription>
					</DialogHeader>
					{deleting.length > 0 && (
						<MedicalRecordDeleteForm
							loading={deletingBusy}
							onSubmit={handleDelete}
						/>
					)}
				</DialogContent>
			</Dialog>

			<EditPrescriptionDialog
				open={showEditPrescription}
				onOpenChange={setShowEditPrescription}
				medicalRecordId={selectedRecord?.ID || 0}
				prescription={selectedRecord?.prescription}
				onSuccess={reloadRecords}
			/>
		</main>
	);

	function handleView(id: number) {
		navigate(`/clinical/medical-records/view/${id}`);
	}

	function handleEdit(id: number) {
		navigate(`/clinical/medical-records/update/${id}`);
	}

	function handleViewPrescription(record: MedicalRecord) {
		setSelectedRecord(record);
		setShowPrescription(true);
	}

	function handleEditPrescription(record: MedicalRecord) {
		setSelectedRecord(record);
		setShowEditPrescription(true);
	}

	async function handleDelete(values: MedicalRecordDeleteValues) {
		if (deleting.length === 0) return;
		const reason = values.reason.trim();
		setDeletingBusy(true);
		const res =
			deleting.length === 1
				? await deleteMedicalRecord(deleting[0].ID, reason)
				: await bulkDeleteMedicalRecords(
						deleting.map((r) => r.ID),
						reason,
					);
		setDeletingBusy(false);
		if (res.success) {
			setDeleting([]);
			// The last records of a page are gone: go back one page.
			if (medicalRecords.length <= deleting.length && page > 1)
				setPage(page - 1);
			else reloadRecords();
		}
	}

	async function handleRestore(record: DeletedMedicalRecord) {
		setRestoringId(record.ID);
		const res = await restoreMedicalRecord(record.ID);
		setRestoringId(null);
		if (res.success) {
			setDeletedLoading(true);
			loadDeleted();
			reloadRecords();
		}
	}

	function reloadRecords() {
		if (!patient) return;
		getMedicalRecords(patient.ID, { page, limit: 10 }).then((res) => {
			if (res.success && res.data) {
				setMedicalRecords(res.data.items);
				setTotalPages(res.data.total_pages);
			}
		});
	}

	function handleSendWhatsAppPrescription(record: MedicalRecord) {
		if (!patient?.phone) {
			infoToast(textGet("clinical.medical_record.contact_error.title"), {
				description: textGet(
					"clinical.medical_record.contact_error.description",
				),
			});
			return;
		}
		const message = buildPrescriptionWhatsAppMessage(
			{
				patientName: `${patient.first_name} ${patient.last_name}`.trim(),
				date: formatDate(record.date),
				items: record.prescription?.items?.map((item) => ({
					medication: item.medication,
					dose: item.dose,
					frequency: item.frequency,
					duration: item.duration,
					notes: item.notes,
				})),
				indications: record.prescription?.indications,
			},
			textGet,
		);
		window.open(generateWhatsAppLink(patient.phone, message), "_blank");
	}

	async function handleDownloadPrescription(record: MedicalRecord) {
		const response = await downloadPrescription(record.ID);
		if (response.success && response.data) {
			const blobUrl = window.URL.createObjectURL(response.data);
			const tempLink = document.createElement("a");
			tempLink.href = blobUrl;
			tempLink.download = response.filename ?? `receta_${record.ID}.pdf`;
			document.body.appendChild(tempLink);
			tempLink.click();
			document.body.removeChild(tempLink);
			window.URL.revokeObjectURL(blobUrl);
		}
	}

	function handleCreateMedicalRecord() {
		const params = new URLSearchParams({
			patient_id: String(patient?.ID),
		});
		if (appointmentId) params.set("appointment_id", appointmentId);
		navigate(`/clinical/medical-records/create?${params.toString()}`);
	}

	function handleEditPatient() {
		navigate(`/clinical/edit/${patient?.ID}`);
	}

	function handleContactWhatsApp() {
		if (!patient?.phone) {
			infoToast(textGet("clinical.medical_record.contact_error.title"), {
				description: textGet(
					"clinical.medical_record.contact_error.description",
				),
			});
			return;
		}
		const url = generateWhatsAppLink(patient.phone, EMPTY_STRING);
		window.open(url, "_blank");
	}

	async function handleCritical() {
		if (!patient) return;

		const { success, data } = await updateCritical(patient.ID);
		if (success && data) {
			setPatient(data as Patient);
		}
	}

	async function handleCriticalRevert() {
		if (!patient) return;

		const { success, data } = await updateCriticalRevert(patient.ID);
		if (success && data) {
			setPatient(data as Patient);
		}
	}
};

export default MedicalRecords;
