import { useText } from "@pengi/shared";
import { Button } from "@pengi/ui";
import React from "react";
import { useNavigate, useSearchParams } from "react-router";
import { getPatientById, type Patient } from "@/api/clinical-service";
import { PageHeader } from "@/components/custom/page-header";
import { RegisterDoctorNotice } from "@/components/features/doctors/doctor-notices";
import { PatientSearch } from "@/components/features/patients/patient-search";
import { patientDisplayName } from "@/lib/exam-orders";
import CreateMedicalRecordForm from "@/sections/forms/clinical/medical-record-create-form";
import VisitTypeChooser from "@/sections/forms/clinical/visit-type-chooser";

const CreateMedicalRecordPage = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const [searchParams, setSearchParams] = useSearchParams();
	const patientId = Number(searchParams.get("patient_id")) || undefined;
	const rawVisitType = searchParams.get("visit_type");
	const visitType =
		rawVisitType === "first" || rawVisitType === "followup"
			? rawVisitType
			: null;

	// Named in the visit-type step, so the doctor sees who they picked.
	const [patient, setPatient] = React.useState<Patient>();
	React.useEffect(() => {
		if (!patientId || visitType) return;
		let cancelled = false;
		getPatientById(patientId).then((res) => {
			if (!cancelled && res.success) setPatient(res.data);
		});
		return () => {
			cancelled = true;
		};
	}, [patientId, visitType]);

	const setParam = (key: string, value: string | null) => {
		const next = new URLSearchParams(searchParams);
		if (value === null) next.delete(key);
		else next.set(key, value);
		setSearchParams(next);
	};

	return (
		<main className="grid grid-cols-1 items-start gap-4">
			<RegisterDoctorNotice />
			{!patientId ? (
				// Started without a patient (the create button): pick one first.
				<div className="mx-auto w-full max-w-xl space-y-4">
					<PageHeader
						title={textGet("clinical.medical_record.new_consultation")}
						description={textGet(
							"clinical.medical_record.pick_patient.description",
						)}
					/>
					<PatientSearch
						onPick={(picked) => setParam("patient_id", String(picked.ID))}
						onCreate={() => navigate("/clinical/create?then=consult")}
					/>
				</div>
			) : visitType ? (
				<CreateMedicalRecordForm visitType={visitType} />
			) : (
				<VisitTypeChooser
					onSelect={(v) => setParam("visit_type", v)}
					subject={
						patient?.ID === patientId && (
							<div className="flex items-center gap-2 rounded-full border bg-card py-1 pr-1 pl-3 text-sm">
								<span className="font-medium">
									{patientDisplayName(patient)}
								</span>
								<span className="text-muted-foreground">
									{patient.document}
								</span>
								<Button
									variant="ghost"
									size="sm"
									className="h-7 rounded-full"
									onClick={() => setParam("patient_id", null)}
								>
									{textGet("clinical.medical_record.pick_patient.change")}
								</Button>
							</div>
						)
					}
				/>
			)}
		</main>
	);
};

export default CreateMedicalRecordPage;
