import {
	Activity,
	ArrowLeft,
	Calendar,
	ClipboardList,
	Download,
	FileText,
	Pill,
	Stethoscope,
} from "lucide-react";
import React from "react";
import { useNavigate, useParams } from "react-router";
import {
	downloadPrescription,
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
import { Spinner } from "@/components/ui/spinner";
import { Text } from "@/components/ui/text";
import { useText } from "@/hooks/use-text";
import { dateParser } from "@/lib/utils";

const ViewMedicalRecord = () => {
	const navigate = useNavigate();
	const { id } = useParams<{ id: string }>();
	const [medicalRecord, setMedicalRecord] =
		React.useState<MedicalRecord | null>(null);
	const [showPrescription, setShowPrescription] = React.useState(false);
	const [isDownloading, setIsDownloading] = React.useState(false);
	const { textGet } = useText();

	const handleDownloadPrescription = async () => {
		if (!id) return;
		setIsDownloading(true);
		const response = await downloadPrescription(Number(id));
		if (response.success && response.data) {
			const blobUrl = window.URL.createObjectURL(response.data);
			const tempLink = document.createElement("a");
			tempLink.href = blobUrl;
			// El nombre exacto nos lo da el backend en Content-Disposition si usamos Axios default y leemos headers,
			// pero desde el frontend es más facil forzar un nombre generico + id
			tempLink.download = `receta_${id}.pdf`;
			document.body.appendChild(tempLink);
			tempLink.click();
			document.body.removeChild(tempLink);
			window.URL.revokeObjectURL(blobUrl);
		}
		setIsDownloading(false);
	};

	React.useEffect(() => {
		if (!id) return;

		getMedicalRecordById(Number(id)).then((res) => {
			if (!res.success) {
				navigate(-1 as unknown as string);
				return;
			}
			if (res.data) {
				setMedicalRecord(res.data as MedicalRecord);
			}
		});
	}, [id, navigate]);

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

			{/* Vital Signs */}
			{medicalRecord.vital_signs && (
				<>
					<Separator />
					<Card className="border-l-4 border-l-rose-500">
						<CardHeader>
							<div className="flex items-center gap-3">
								<div className="flex h-9 w-9 items-center justify-center rounded-lg bg-rose-500 text-primary-foreground">
									<Activity className="h-4 w-4" />
								</div>
								<div>
									<CardTitle className="text-lg">
										<Text uuid="view.medical_record.vital_signs.title" />
									</CardTitle>
									<CardDescription>
										<Text uuid="view.medical_record.vital_signs.description" />
									</CardDescription>
								</div>
							</div>
						</CardHeader>
						<CardContent>
							<div className="grid grid-cols-2 md:grid-cols-3 gap-4 text-sm">
								{medicalRecord.vital_signs.weight != null && (
									<div className="space-y-1">
										<p className="text-muted-foreground font-medium">{textGet("view.medical_record.vital_signs.weight")}</p>
										<p className="font-semibold">{medicalRecord.vital_signs.weight} kg</p>
									</div>
								)}
								{medicalRecord.vital_signs.height != null && (
									<div className="space-y-1">
										<p className="text-muted-foreground font-medium">{textGet("view.medical_record.vital_signs.height")}</p>
										<p className="font-semibold">{medicalRecord.vital_signs.height} cm</p>
									</div>
								)}
								{medicalRecord.vital_signs.blood_pressure && (
									<div className="space-y-1">
										<p className="text-muted-foreground font-medium">{textGet("view.medical_record.vital_signs.blood_pressure")}</p>
										<p className="font-semibold">{medicalRecord.vital_signs.blood_pressure} mmHg</p>
									</div>
								)}
								{medicalRecord.vital_signs.temperature != null && (
									<div className="space-y-1">
										<p className="text-muted-foreground font-medium">{textGet("view.medical_record.vital_signs.temperature")}</p>
										<p className="font-semibold">{medicalRecord.vital_signs.temperature} °C</p>
									</div>
								)}
								{medicalRecord.vital_signs.heart_rate != null && (
									<div className="space-y-1">
										<p className="text-muted-foreground font-medium">{textGet("view.medical_record.vital_signs.heart_rate")}</p>
										<p className="font-semibold">{medicalRecord.vital_signs.heart_rate} bpm</p>
									</div>
								)}
								{medicalRecord.vital_signs.o2_saturation != null && (
									<div className="space-y-1">
										<p className="text-muted-foreground font-medium">{textGet("view.medical_record.vital_signs.o2_saturation")}</p>
										<p className="font-semibold">{medicalRecord.vital_signs.o2_saturation}%</p>
									</div>
								)}
							</div>
						</CardContent>
					</Card>
				</>
			)}

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
						<CardContent className="space-y-4">
							{(medicalRecord.prescription.items?.length ?? 0) > 0 && (
								<div className="space-y-2">
									<p className="text-sm font-medium">
										<Text uuid="view.medical_record.prescription.items" />
									</p>
									<div className="divide-y rounded-md border text-sm">
										{medicalRecord.prescription.items?.map((item) => (
											<div key={item.ID} className="grid grid-cols-2 md:grid-cols-4 gap-2 p-3">
												<div>
													<span className="text-muted-foreground block text-xs">{textGet("view.medical_record.prescription.item.medication")}</span>
													<span className="font-medium">{item.medication}</span>
												</div>
												<div>
													<span className="text-muted-foreground block text-xs">{textGet("view.medical_record.prescription.item.dose")}</span>
													<span>{item.dose}</span>
												</div>
												<div>
													<span className="text-muted-foreground block text-xs">{textGet("view.medical_record.prescription.item.frequency")}</span>
													<span>{item.frequency}</span>
												</div>
												<div>
													<span className="text-muted-foreground block text-xs">{textGet("view.medical_record.prescription.item.duration")}</span>
													<span>{item.duration}</span>
												</div>
												{item.notes && (
													<div className="col-span-2 md:col-span-4">
														<span className="text-muted-foreground block text-xs">{textGet("view.medical_record.prescription.item.notes")}</span>
														<span className="text-muted-foreground">{item.notes}</span>
													</div>
												)}
											</div>
										))}
									</div>
								</div>
							)}
							<div className="flex flex-wrap gap-2">
								<Button
									variant="outline"
									onClick={() => setShowPrescription(true)}
								>
									<Pill className="h-4 w-4 mr-2" />
									<Text uuid="view.medical_record.prescription.view" />
								</Button>
								<Button
									variant="default"
									onClick={handleDownloadPrescription}
									disabled={isDownloading}
								>
									{isDownloading ? (
										<Spinner className="mr-2 h-4 w-4" />
									) : (
										<Download className="mr-2 h-4 w-4" />
									)}
									<Text uuid="view.medical_record.prescription.download" />
								</Button>
							</div>
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
