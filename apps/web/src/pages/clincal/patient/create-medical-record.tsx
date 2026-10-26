import { useText } from "@pengi/shared";
import { useSearchParams } from "react-router";
import { PageHeader } from "@/components/custom/page-header";
import { RegisterDoctorNotice } from "@/components/features/doctors/doctor-notices";
import { PatientSearch } from "@/components/features/patients/patient-search";
import CreateMedicalRecordForm from "@/sections/forms/clinical/medical-record-create-form";
import VisitTypeChooser from "@/sections/forms/clinical/visit-type-chooser";

const CreateMedicalRecordPage = () => {
	const { textGet } = useText();
	const [searchParams, setSearchParams] = useSearchParams();
	const hasPatient = !!searchParams.get("patient_id");
	const rawVisitType = searchParams.get("visit_type");
	const visitType =
		rawVisitType === "first" || rawVisitType === "followup"
			? rawVisitType
			: null;

	const setParam = (key: string, value: string) => {
		const next = new URLSearchParams(searchParams);
		next.set(key, value);
		setSearchParams(next);
	};

	return (
		<main className="grid grid-cols-1 items-start gap-4">
			<RegisterDoctorNotice />
			{!hasPatient ? (
				// Started without a patient (the phone's create button): pick one first.
				<div className="mx-auto w-full max-w-xl space-y-4">
					<PageHeader
						title={textGet("clinical.medical_record.new_consultation")}
						description={textGet(
							"clinical.medical_record.pick_patient.description",
						)}
					/>
					<PatientSearch
						onPick={(patient) => setParam("patient_id", String(patient.ID))}
					/>
				</div>
			) : visitType ? (
				<CreateMedicalRecordForm visitType={visitType} />
			) : (
				<VisitTypeChooser onSelect={(v) => setParam("visit_type", v)} />
			)}
		</main>
	);
};

export default CreateMedicalRecordPage;
