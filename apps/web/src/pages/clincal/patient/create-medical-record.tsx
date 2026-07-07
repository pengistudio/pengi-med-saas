import { useSearchParams } from "react-router";
import CreateMedicalRecordForm from "@/sections/forms/clinical/medical-record-create-form";
import VisitTypeChooser from "@/sections/forms/clinical/visit-type-chooser";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const CreateMedicalRecordPage = () => {
	const [searchParams, setSearchParams] = useSearchParams();
	const rawVisitType = searchParams.get("visit_type");
	const visitType =
		rawVisitType === "first" || rawVisitType === "followup"
			? rawVisitType
			: null;

	return (
		<DashboardLayout>
			<main className="grid items-start gap-4 p-4 sm:px-6 sm:py-0">
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
		</DashboardLayout>
	);
};

export default CreateMedicalRecordPage;
