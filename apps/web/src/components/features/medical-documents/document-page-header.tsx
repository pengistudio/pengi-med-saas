import { useText } from "@pengi/shared";
import { Button, Text } from "@pengi/ui";
import { ArrowLeft } from "lucide-react";
import { useNavigate } from "react-router";
import type { Patient } from "@/api/clinical-service";

function ageInYears(birthDate: string | undefined): number | null {
	if (!birthDate) return null;
	const birth = new Date(birthDate);
	if (Number.isNaN(birth.getTime())) return null;
	const now = new Date();
	let age = now.getFullYear() - birth.getFullYear();
	const beforeBirthday =
		now.getMonth() < birth.getMonth() ||
		(now.getMonth() === birth.getMonth() && now.getDate() < birth.getDate());
	if (beforeBirthday) age--;
	return age >= 0 ? age : null;
}

interface DocumentPageHeaderProps {
	patientId: number;
	patient: Patient | null;
	title: string;
	description: string;
}

/** Back link, title and the patient the document is about. */
export function DocumentPageHeader({
	patientId,
	patient,
	title,
	description,
}: DocumentPageHeaderProps) {
	const navigate = useNavigate();
	const { textGet, formatDate } = useText();
	const fullName = patient
		? patient.full_name || `${patient.first_name} ${patient.last_name}`
		: "";
	const age = ageInYears(patient?.birth_date);

	return (
		<header className="space-y-4">
			<Button
				type="button"
				variant="ghost"
				size="sm"
				className="-ml-2 w-fit"
				onClick={() =>
					navigate(`/clinical/medical-documents?patient_id=${patientId}`)
				}
			>
				<ArrowLeft className="mr-2 size-4" />
				<Text uuid="clinical.medical_documents.back" />
			</Button>
			<div className="space-y-1">
				<h1 className="text-2xl font-semibold tracking-tight">
					<Text uuid={title} />
				</h1>
				<p className="text-sm text-muted-foreground">
					<Text uuid={description} />
				</p>
			</div>
			{patient && (
				<dl className="flex flex-wrap gap-x-8 gap-y-3 rounded-xl border bg-card px-5 py-4 text-sm">
					<div>
						<dt className="text-xs text-muted-foreground">
							<Text uuid="clinical.patient.name" />
						</dt>
						<dd className="text-base font-medium">{fullName}</dd>
					</div>
					<div>
						<dt className="text-xs text-muted-foreground">
							<Text uuid="clinical.patient.document" />
						</dt>
						<dd className="font-medium tabular-nums">{patient.document}</dd>
					</div>
					{age !== null && (
						<div>
							<dt className="text-xs text-muted-foreground">
								<Text uuid="medical_document.age" />
							</dt>
							<dd className="font-medium tabular-nums">
								{age} {textGet("medical_document.years")}
							</dd>
						</div>
					)}
					<div>
						<dt className="text-xs text-muted-foreground">
							<Text uuid="dialog.medical_report.created_at" />
						</dt>
						<dd className="font-medium tabular-nums">
							{formatDate(new Date())}
						</dd>
					</div>
				</dl>
			)}
		</header>
	);
}
