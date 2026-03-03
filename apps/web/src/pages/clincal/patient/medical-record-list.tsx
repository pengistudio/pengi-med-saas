import {
	CalendarCheck,
	CircleAlert,
	Pencil,
	Triangle,
	TriangleAlert,
} from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import {
	downloadPrescription,
	getMedicalRecords,
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
import { getMedicalRecordColumns } from "@/sections/columns/clinical/medical-record-columns";
import { DashboardLayout } from "@/sections/template/dashboard-template";
import { usePatientStore } from "@/store/patient-store";

const MedicalRecords = () => {
	const { patient, setPatient } = usePatientStore();
	const navigate = useNavigate();
	const { errorToast, infoToast } = useToast();
	const { checkPermission } = usePermission();
	const { textGet } = useText();

	const [medicalRecords, setMedicalRecords] = React.useState<MedicalRecord[]>(
		[],
	);
	const [loading, setLoading] = React.useState(true);
	const [showPrescription, setShowPrescription] = React.useState(false);
	const [showEditPrescription, setShowEditPrescription] = React.useState(false);
	const [selectedRecord, setSelectedRecord] =
		React.useState<MedicalRecord | null>(null);

	React.useEffect(() => {
		if (!patient) return;

		getMedicalRecords(patient.ID)
			.then((res) => {
				const { success, data, message } = res;
				if (!success) {
					errorToast(null, message);
					return;
				}
				if (data) {
					setMedicalRecords(data as MedicalRecord[]);
				}
			})
			.catch(() => {
				infoToast(textGet("error.unexpected.title"), {
					description: textGet("error.unexpected.description"),
				});
			})
			.finally(() => {
				setLoading(false);
			});
	}, [patient, errorToast, infoToast, textGet]);

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
				<div className="grid lg:grid-cols-[400px_1fr] md:grid-cols-[300px_1fr] grid-cols-1 gap-4">
					<div className="space-y-4">
						{patient && <PatientCard patient={patient} />}
						<Card>
							<CardHeader>
								<CardTitle className="flex items-center gap-2">
									<Triangle className="h-5 w-5" />
									<Text uuid="clinical.medical_record.actions" />
								</CardTitle>
							</CardHeader>
							<CardContent className="grid grid-cols-1 gap-4">
								{checkPermission([
									PERMISSIONS.MEDICAL_RECORD.PERMISSION_CREATE_MEDICAL_RECORD,
								]) && (
									<Button variant="outline" onClick={handleCreateMedicalRecord}>
										<Text uuid="clinical.medical_record.new_consultation" />
									</Button>
								)}
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
						{loading ? (
							<div className="h-24 w-full animate-pulse bg-muted rounded-md" />
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
								)}
								data={medicalRecords}
							/>
						)}
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
		getMedicalRecords(patient.ID).then((res) => {
			if (res.success && res.data) {
				setMedicalRecords(res.data as MedicalRecord[]);
			}
		});
	}

	async function handleDownloadPrescription(record: MedicalRecord) {
		const response = await downloadPrescription(record.ID);
		if (response.success && response.data) {
			const blobUrl = window.URL.createObjectURL(response.data);
			const tempLink = document.createElement("a");
			tempLink.href = blobUrl;
			tempLink.download = `receta_${record.ID}.pdf`;
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

		const { success, data, message } = await updateCritical(patient.ID);
		if (!success) {
			errorToast(null, message);
			return;
		}
		if (data) {
			setPatient(data as Patient);
		}
	}

	async function handleCriticalRevert() {
		if (!patient) return;

		const { success, data, message } = await updateCriticalRevert(patient.ID);
		if (!success) {
			errorToast(null, message);
			return;
		}
		if (data) {
			setPatient(data as Patient);
		}
	}
};

function generateWhatsAppLink(phoneNumber: string, message?: string): string {
	let normalizedNumber = phoneNumber.trim();
	normalizedNumber = normalizedNumber.replace(/[^0-9+]/g, "");

	if (normalizedNumber.startsWith("0")) {
		normalizedNumber = `593${normalizedNumber.slice(1)}`;
	}

	if (normalizedNumber.startsWith("+")) {
		normalizedNumber = normalizedNumber.slice(1);
	}

	const baseUrl = `https://wa.me/${normalizedNumber}`;
	return message ? `${baseUrl}?text=${encodeURIComponent(message)}` : baseUrl;
}

export default MedicalRecords;
