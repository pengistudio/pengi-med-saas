import {
	CalendarCheck,
	CircleAlert,
	Pencil,
	Plus,
	Stethoscope,
	TriangleAlert,
} from "lucide-react";
import React from "react";
import { useNavigate, useParams } from "react-router";
import {
	downloadPrescription,
	getMedicalRecords,
	getPatientById,
	type MedicalRecord,
	type Patient,
	updateCritical,
	updateCriticalRevert,
} from "@/api/clinical-service";
import PatientCard from "@/components/custom/patient-card";
import { DataTable } from "@/components/custom/table/data-table";
import EditPrescriptionDialog from "@/components/features/patient/edit-prescription-dialog";
import PrescriptionDialog from "@/components/features/patient/prescription-dialog";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Text } from "@/components/ui/text";
import usePermission from "@/hooks/use-permission";
import { useText } from "@/hooks/use-text";
import useToast from "@/hooks/use-toast";
import { EMPTY_STRING, PERMISSIONS } from "@/lib/constants";
import {
	buildPrescriptionWhatsAppMessage,
	dateParser,
	generateWhatsAppLink,
} from "@/lib/utils";
import { getMedicalRecordColumns } from "@/sections/columns/clinical/medical-record-columns";
import { DashboardLayout } from "@/sections/template/dashboard-template";
import {
	selectPatient,
	selectSetPatient,
	usePatientStore,
} from "@/store/patient-store";

const MedicalRecords = () => {
	const patient = usePatientStore(selectPatient);
	const setPatient = usePatientStore(selectSetPatient);
	const { patientId } = useParams<{ patientId: string }>();
	const navigate = useNavigate();

	React.useEffect(() => {
		if (!patientId) return;
		getPatientById(Number(patientId)).then((res) => {
			if (res.success && res.data) setPatient(res.data as Patient);
		});
	}, [patientId, setPatient]);
	const { infoToast } = useToast();
	const { checkPermission } = usePermission();
	const { textGet } = useText();

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

	React.useEffect(() => {
		if (!patient) return;
		setLoading(true);
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

	return (
		<DashboardLayout>
			<main className="grid flex-1 items-start gap-4 p-4 sm:px-6 sm:py-0 md:gap-8 auto-rows-max">
				{patient && (
					<nav className="flex items-center gap-1.5 text-sm text-muted-foreground">
						<button
							type="button"
							onClick={() => navigate("/clinical")}
							className="hover:text-foreground transition-colors"
						>
							<Text uuid="dashboard.clinical.patients" />
						</button>
						<span>/</span>
						<span className="text-foreground font-medium">
							{`${patient.last_name} ${patient.first_name}`.trim()}
						</span>
					</nav>
				)}
				<div className="grid lg:grid-cols-[400px_1fr] md:grid-cols-[300px_1fr] grid-cols-1 gap-4">
					<div className="space-y-4">
						{patient && (
							<PatientCard
								patient={patient}
								onEditPatient={handleEditPatient}
								headerAction={
									checkPermission([
										PERMISSIONS.MEDICAL_RECORD.PERMISSION_CREATE_MEDICAL_RECORD,
									]) ? (
										<Button size="sm" onClick={handleCreateMedicalRecord}>
											<Plus className="w-3.5 h-3.5 mr-1.5" />
											<Text uuid="clinical.medical_record.new_consultation" />
										</Button>
									) : undefined
								}
							/>
						)}
						<Card>
							<CardHeader>
								<CardTitle className="flex items-center gap-2">
									<Stethoscope className="h-5 w-5" />
									<Text uuid="clinical.medical_record.actions" />
								</CardTitle>
							</CardHeader>
							<CardContent className="grid grid-cols-1 gap-4">
								<Button
									variant="outline"
									onClick={() => setShowPrescription(true)}
									disabled={!latestPrescription}
								>
									<Text uuid="clinical.medical_record.view_prescriptions" />
								</Button>
								{checkPermission([
									PERMISSIONS.MEDICAL_RECORD.PERMISSION_UPDATE_PATIENT,
								]) && (
									<Button variant="outline" onClick={handleEditPatient}>
										<Pencil className="w-4 h-4 mr-2" />
										<Text uuid="clinical.medical_record.edit_patient" />
									</Button>
								)}
								<Button variant="outline" onClick={handleContactWhatsApp}>
									<Text uuid="clinical.medical_record.contact_ws" />
								</Button>
								<Button
									variant={patient?.critical ? "outline" : "destructive"}
									onClick={
										patient?.critical ? handleCriticalRevert : handleCritical
									}
								>
									{patient?.critical ? (
										<CircleAlert className="w-4 h-4 mr-2" />
									) : (
										<TriangleAlert className="w-4 h-4 mr-2" />
									)}
									{patient?.critical ? (
										<Text uuid="clinical.medical_record.unmark_critical" />
									) : (
										<Text uuid="clinical.medical_record.mark_critical" />
									)}
								</Button>
							</CardContent>
						</Card>
					</div>
					<div className="sm:max-w-[calc(100vw-6.5rem)] max-w-[calc(100vw-2rem)]">
						<h2 className="flex flex-row items-center justify-start text-xl gap-2">
							<CalendarCheck className="w-5 h-5 text-muted-foreground" />
							<Text uuid="clinical.medical_record.history_title" />
						</h2>
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
							)}
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
					</div>
				</div>

				<PrescriptionDialog
					open={showPrescription}
					onOpenChange={setShowPrescription}
					prescription={selectedRecord?.prescription || latestPrescription}
				/>

				<EditPrescriptionDialog
					open={showEditPrescription}
					onOpenChange={setShowEditPrescription}
					medicalRecordId={selectedRecord?.ID || 0}
					prescription={selectedRecord?.prescription}
					onSuccess={reloadRecords}
				/>
			</main>
		</DashboardLayout>
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
		const message = buildPrescriptionWhatsAppMessage({
			patientName: `${patient.first_name} ${patient.last_name}`.trim(),
			date: dateParser(new Date(record.date), { dateStyle: "medium" }),
			items: record.prescription?.items?.map((item) => ({
				medication: item.medication,
				dose: item.dose,
				frequency: item.frequency,
				duration: item.duration,
				notes: item.notes,
			})),
			indications: record.prescription?.indications,
		});
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
