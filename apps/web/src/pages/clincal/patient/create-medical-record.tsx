import { useSearchParams } from "react-router";
import { RegisterDoctorNotice } from "@/components/features/doctors/doctor-notices";
import CreateMedicalRecordForm from "@/sections/forms/clinical/medical-record-create-form";
import VisitTypeChooser from "@/sections/forms/clinical/visit-type-chooser";

const CreateMedicalRecordPage = () => {
	const [searchParams, setSearchParams] = useSearchParams();
	const rawVisitType = searchParams.get("visit_type");
	const visitType =
		rawVisitType === "first" || rawVisitType === "followup"
			? rawVisitType
			: null;

	return (
		<main className="grid grid-cols-1 items-start gap-4">
			<RegisterDoctorNotice />
			{visitType ? (
				<CreateMedicalRecordForm visitType={visitType} />
			) : (
				<VisitTypeChooser
					onSelect={(v) => {
						const next = new URLSearchParams(searchParams);
						next.set("visit_type", v);
						setSearchParams(next);
					}}
				/>
			)}
		</main>
	);
};

export default CreateMedicalRecordPage;
