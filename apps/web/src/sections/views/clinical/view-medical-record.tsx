import {
	ArrowLeft,
	Calendar,
	ClipboardList,
	FileText,
	Pill,
	Stethoscope,
} from "lucide-react";
import React from "react";
import { useNavigate, useParams } from "react-router";
import {
	getMedicalRecordById,
	type MedicalRecord,
} from "@/api/clinical-service";
import PrescriptionDialog from "@/components/features/patient/prescription-dialog";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Text } from "@/components/ui/text";
import { useText } from "@/hooks/use-text";
import useToast from "@/hooks/use-toast";
import { dateParser } from "@/lib/utils";

const ViewMedicalRecord = () => {
	const navigate = useNavigate();
	const { id } = useParams<{ id: string }>();
	const [medicalRecord, setMedicalRecord] =
		React.useState<MedicalRecord | null>(null);
	const [showPrescription, setShowPrescription] = React.useState(false);
	const { errorToast } = useToast();
	const { textGet } = useText();

	React.useEffect(() => {
		if (!id) return;

		getMedicalRecordById(Number(id)).then((res) => {
			if (!res.success) {
				errorToast(null, res.message);
				navigate(-1 as unknown as string);
				return;
			}
			if (res.data) {
				setMedicalRecord(res.data as MedicalRecord);
			}
		});
	}, [id, errorToast, navigate]);

	if (!medicalRecord) {
		return (
			<div className="flex items-center justify-center h-64">
				<p className="text-muted-foreground">
					<Text uuid="view.medical_record.loading" />
				</p>
			</div>
		);
	}

	return (
		<div className="space-y-4 max-w-4xl mx-auto w-full">
			{/* Header */}
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-3xl font-bold">
						<Text uuid="view.medical_record.title" />
					</h1>
					<p className="text-muted-foreground">
						<Text uuid="view.medical_record.subtitle" />
					</p>
				</div>
				<Button
					variant="outline"
					onClick={() => navigate(-1 as unknown as string)}
				>
					<ArrowLeft className="mr-2 h-4 w-4" />
					<Text uuid="view.medical_record.back" />
				</Button>
			</div>

			{/* Basic Info Card */}
			<Card>
				<CardHeader>
					<CardTitle>
						<Text uuid="view.medical_record.info.title" />
					</CardTitle>
					<CardDescription>
						<Text uuid="view.medical_record.info.description" />
					</CardDescription>
				</CardHeader>
				<CardContent className="space-y-4">
					<div className="grid gap-3 text-sm">
						<div className="grid grid-cols-[120px_1fr] items-center">
							<span className="font-medium text-muted-foreground">
								{textGet("view.medical_record.date")}:
							</span>
							<span>
								{dateParser(new Date(medicalRecord.date), {
									dateStyle: "full",
								})}
							</span>
						</div>
						<div className="grid grid-cols-[120px_1fr] items-start">
							<span className="font-medium text-muted-foreground">
								{textGet("view.medical_record.motive")}:
							</span>
							<span className="wrap-break-word">{medicalRecord.motive}</span>
						</div>
						{medicalRecord.observation && (
							<div className="grid grid-cols-[120px_1fr] items-start">
								<span className="font-medium text-muted-foreground">
									{textGet("view.medical_record.observation")}:
								</span>
								<span className="wrap-break-word whitespace-pre-wrap">
									{medicalRecord.observation}
								</span>
							</div>
						)}
					</div>
				</CardContent>
			</Card>

			{/* SOAP: Subjective */}
			<Card className="border-l-4 border-l-primary">
				<CardHeader>
					<div className="flex items-center gap-3">
						<div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary text-primary-foreground">
							<FileText className="h-4 w-4" />
						</div>
						<div>
							<CardTitle className="text-lg">
								<Text uuid="view.medical_record.soap.subjective.title" />
							</CardTitle>
							<p className="text-sm text-muted-foreground">
								<Text uuid="view.medical_record.soap.subjective.description" />
							</p>
						</div>
					</div>
				</CardHeader>
				<CardContent>
					<div className="bg-muted/50 rounded-lg p-4">
						<p className="text-sm whitespace-pre-wrap">
							{medicalRecord.soap_record?.subjective ||
								textGet("view.medical_record.not_registered")}
						</p>
					</div>
				</CardContent>
			</Card>

			{/* SOAP: Objective */}
			<Card className="border-l-4 border-l-emerald-400">
				<CardHeader>
					<div className="flex items-center gap-3">
						<div className="flex h-9 w-9 items-center justify-center rounded-lg bg-emerald-400 text-primary-foreground">
							<Stethoscope className="h-4 w-4" />
						</div>
						<div>
							<CardTitle className="text-lg">
								<Text uuid="view.medical_record.soap.objective.title" />
							</CardTitle>
							<p className="text-sm text-muted-foreground">
								<Text uuid="view.medical_record.soap.objective.description" />
							</p>
						</div>
					</div>
				</CardHeader>
				<CardContent>
					<div className="bg-muted/50 rounded-lg p-4">
						<p className="text-sm whitespace-pre-wrap">
							{medicalRecord.soap_record?.objective ||
								textGet("view.medical_record.not_registered")}
						</p>
					</div>
				</CardContent>
			</Card>

			{/* SOAP: Assessment */}
			<Card className="border-l-4 border-l-orange-500">
				<CardHeader>
					<div className="flex items-center gap-3">
						<div className="flex h-9 w-9 items-center justify-center rounded-lg bg-orange-500 text-primary-foreground">
							<ClipboardList className="h-4 w-4" />
						</div>
						<div>
							<CardTitle className="text-lg">
								<Text uuid="view.medical_record.soap.assessment.title" />
							</CardTitle>
							<p className="text-sm text-muted-foreground">
								<Text uuid="view.medical_record.soap.assessment.description" />
							</p>
						</div>
					</div>
				</CardHeader>
				<CardContent>
					<div className="bg-muted/50 rounded-lg p-4">
						<p className="text-sm whitespace-pre-wrap">
							{medicalRecord.soap_record?.assessment ||
								textGet("view.medical_record.not_registered")}
						</p>
					</div>
				</CardContent>
			</Card>

			{/* SOAP: Plan */}
			<Card className="border-l-4 border-l-teal-400">
				<CardHeader>
					<div className="flex items-center gap-3">
						<div className="flex h-9 w-9 items-center justify-center rounded-lg bg-teal-400 text-primary-foreground">
							<Calendar className="h-4 w-4" />
						</div>
						<div>
							<CardTitle className="text-lg">
								<Text uuid="view.medical_record.soap.plan.title" />
							</CardTitle>
							<p className="text-sm text-muted-foreground">
								<Text uuid="view.medical_record.soap.plan.description" />
							</p>
						</div>
					</div>
				</CardHeader>
				<CardContent>
					<div className="bg-muted/50 rounded-lg p-4">
						<p className="text-sm whitespace-pre-wrap">
							{medicalRecord.soap_record?.plan ||
								textGet("view.medical_record.not_registered")}
						</p>
					</div>
				</CardContent>
			</Card>

			{/* Prescription Section */}
			{medicalRecord.prescription && (
				<>
					<Separator />
					<Card className="border-l-4 border-l-violet-500">
						<CardHeader>
							<div className="flex items-center gap-3">
								<div className="flex h-9 w-9 items-center justify-center rounded-lg bg-violet-500 text-primary-foreground">
									<Pill className="h-4 w-4" />
								</div>
								<div>
									<CardTitle className="text-lg">
										<Text uuid="view.medical_record.prescription.title" />
									</CardTitle>
									<CardDescription>
										<Text uuid="view.medical_record.prescription.description" />
									</CardDescription>
								</div>
							</div>
						</CardHeader>
						<CardContent>
							<Button
								variant="outline"
								onClick={() => setShowPrescription(true)}
							>
								<Pill className="h-4 w-4 mr-2" />
								<Text uuid="view.medical_record.prescription.view" />
							</Button>
						</CardContent>
					</Card>
				</>
			)}

			<PrescriptionDialog
				open={showPrescription}
				onOpenChange={setShowPrescription}
				prescription={medicalRecord.prescription}
			/>
		</div>
	);
};

export default ViewMedicalRecord;
