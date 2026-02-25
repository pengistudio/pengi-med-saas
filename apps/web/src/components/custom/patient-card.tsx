import {
	AlertCircle,
	FileText,
	Heart,
	Phone,
	Pill,
	Stethoscope,
	UserRound,
} from "lucide-react";
import React from "react";
import type { Patient } from "@/api/clinical-service";
import PrescriptionDialog from "@/components/features/patient/prescription-dialog";
import {
	Accordion,
	AccordionContent,
	AccordionItem,
	AccordionTrigger,
} from "@/components/ui/accordion";
import { Badge } from "@/components/ui/badge";
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
import { dateParser } from "@/lib/utils";

/**
 * Extends `Patient` with optional fields that may be injected
 * by the view layer (e.g. latest prescription, family contacts).
 */
interface PatientCardData extends Patient {
	full_name?: string;
	family_number?: string;
	second_family_number?: string;
	family_members_count?: number;
	latest_prescription?: {
		content: string;
		indications: string;
	} | null;
}

const NotAvailable = () => (
	<span className="text-muted-foreground italic text-sm">
		<Text uuid="clinical.patient_card.not_available" />
	</span>
);

export default function PatientCard({ patient }: { patient: PatientCardData }) {
	const [showPrescription, setShowPrescription] = React.useState(false);
	const { textGet } = useText();

	const displayName =
		patient.full_name || `${patient.first_name} ${patient.last_name}`;

	return (
		<>
			<Card className="h-fit border-l-4 border-l-primary">
				<CardHeader className="space-y-4">
					<div className="flex items-start justify-between gap-4">
						<div className="space-y-2">
							<CardTitle className="text-2xl leading-tight wrap-break-word max-w-[250px]">
								{displayName}
							</CardTitle>
							<CardDescription className="flex items-center gap-2">
								<FileText className="h-4 w-4 shrink-0" />
								<Text uuid="clinical.patient_card.document" />{" "}
								<span className="text-foreground">{patient.document}</span>
							</CardDescription>
						</div>
						{patient.critical && (
							<Badge
								variant="destructive"
								className="px-4 py-1 shrink-0 hover:cursor-default"
							>
								<AlertCircle className="h-4 w-4 mr-1" />
								<Text uuid="clinical.patient_card.critical" />
							</Badge>
						)}
					</div>
				</CardHeader>
				<CardContent className="space-y-6">
					<div className="space-y-4">
						<h3 className="font-semibold flex items-center gap-2 text-sm text-muted-foreground">
							<UserRound className="h-4 w-4" />
							<Text uuid="clinical.patient_card.personal_info" />
						</h3>
						<div className="grid gap-3 text-sm">
							<div className="grid grid-cols-[120px_1fr] items-center">
								<span className="font-medium text-muted-foreground">
									<Text uuid="clinical.patient_card.birth_date" />
								</span>
								<span>
									{dateParser(patient.birth_date, {
										dateStyle: "medium",
									})}
								</span>
							</div>
							<div className="grid grid-cols-[120px_1fr] items-center">
								<span className="font-medium text-muted-foreground">
									<Text uuid="clinical.patient_card.gender" />
								</span>
								<span>{formatGender(patient.gender, textGet)}</span>
							</div>
							<div className="grid grid-cols-[120px_1fr] items-center">
								<span className="font-medium text-muted-foreground">
									<Text uuid="clinical.patient_card.phone" />
								</span>
								<div className="flex items-center gap-2">
									{patient.phone ? (
										<span>{patient.phone}</span>
									) : (
										<Badge variant="secondary" className="font-normal">
											<Phone className="h-3 w-3 mr-1" />
											<Text uuid="clinical.patient_card.not_registered" />
										</Badge>
									)}
								</div>
							</div>
							{patient.family_number && (
								<div className="grid grid-cols-[120px_1fr] items-center">
									<span className="font-medium text-muted-foreground">
										<Text uuid="clinical.patient_card.family_phone" />
									</span>
									<span>{patient.family_number}</span>
								</div>
							)}
							{patient.second_family_number && (
								<div className="grid grid-cols-[120px_1fr] items-center">
									<span className="font-medium text-muted-foreground">
										<Text uuid="clinical.patient_card.family_phone_2" />
									</span>
									<span>{patient.second_family_number}</span>
								</div>
							)}
						</div>
					</div>

					<Separator />

					<div className="space-y-4">
						<h3 className="font-semibold flex items-center gap-2 text-sm text-muted-foreground">
							<Stethoscope className="h-4 w-4" />
							<Text uuid="clinical.patient_card.medical_info" />
						</h3>
						<div className="grid gap-3 text-sm">
							<div className="grid grid-cols-[120px_1fr] items-center">
								<span className="font-medium text-muted-foreground">
									<Text uuid="clinical.patient_card.insurance" />
								</span>
								{patient.insurance ? (
									<span>{patient.insurance}</span>
								) : (
									<NotAvailable />
								)}
							</div>
							<div className="grid grid-cols-[120px_1fr] items-center">
								<span className="font-medium text-muted-foreground">
									<Text uuid="clinical.patient_card.medic" />
								</span>
								{patient.medic ? (
									<span>{patient.medic}</span>
								) : (
									<NotAvailable />
								)}
							</div>
							<div className="grid grid-cols-[120px_1fr] items-start">
								<span className="font-medium text-muted-foreground">
									<Text uuid="clinical.patient_card.diagnosis" />
								</span>
								{patient.diagnosis ? (
									<span className="wrap-break-word">{patient.diagnosis}</span>
								) : (
									<NotAvailable />
								)}
							</div>
						</div>
					</div>

					{patient.latest_prescription && (
						<>
							<Separator />

							<div className="space-y-4">
								<h3 className="font-semibold flex items-center gap-2 text-sm text-muted-foreground">
									<Pill className="h-4 w-4" />
									<Text uuid="clinical.patient_card.latest_prescription" />
								</h3>
								<Button
									variant="outline"
									size="sm"
									onClick={() => setShowPrescription(true)}
									className="w-full"
								>
									<Pill className="h-4 w-4 mr-2" />
									<Text uuid="clinical.patient_card.view_prescription" />
								</Button>
							</div>
						</>
					)}

					{hasAnyMedicalHistory() && (
						<>
							<Separator />

							<div className="space-y-4">
								<h3 className="font-semibold flex items-center gap-2 text-sm text-muted-foreground">
									<FileText className="h-4 w-4" />
									<Text uuid="clinical.patient_card.medical_history" />
								</h3>

								<Accordion multiple className="w-full">
									{patient.app?.trim() && (
										<AccordionItem value="app">
											<AccordionTrigger className="text-sm hover:no-underline">
												<span className="font-medium">
													<Text uuid="clinical.patient_card.app" />
												</span>
											</AccordionTrigger>
											<AccordionContent>
												<p className="text-sm text-muted-foreground whitespace-pre-wrap bg-muted p-3 rounded-md">
													{patient.app}
												</p>
											</AccordionContent>
										</AccordionItem>
									)}

									{patient.apf?.trim() && (
										<AccordionItem value="apf">
											<AccordionTrigger className="text-sm hover:no-underline">
												<span className="font-medium">
													<Text uuid="clinical.patient_card.apf" />
												</span>
											</AccordionTrigger>
											<AccordionContent>
												<p className="text-sm text-muted-foreground whitespace-pre-wrap bg-muted p-3 rounded-md">
													{patient.apf}
												</p>
											</AccordionContent>
										</AccordionItem>
									)}

									{patient.apqx?.trim() && (
										<AccordionItem value="apqx">
											<AccordionTrigger className="text-sm hover:no-underline">
												<span className="font-medium">
													<Text uuid="clinical.patient_card.apqx" />
												</span>
											</AccordionTrigger>
											<AccordionContent>
												<p className="text-sm text-muted-foreground whitespace-pre-wrap bg-muted p-3 rounded-md">
													{patient.apqx}
												</p>
											</AccordionContent>
										</AccordionItem>
									)}
								</Accordion>
							</div>
						</>
					)}

					<Separator />

					<div className="space-y-4">
						<h3 className="font-semibold flex items-center gap-2 text-sm text-muted-foreground">
							<Heart className="h-4 w-4" />
							<Text uuid="clinical.patient_card.notes" />
						</h3>
						{patient.notes ? (
							<p className="text-sm">{patient.notes}</p>
						) : (
							<div className="text-center py-3 rounded-md">
								<span className="text-sm text-muted-foreground">
									<Text uuid="clinical.patient_card.no_notes" />
								</span>
							</div>
						)}
					</div>
				</CardContent>
			</Card>

			<PrescriptionDialog
				open={showPrescription}
				onOpenChange={setShowPrescription}
				prescription={patient.latest_prescription}
			/>
		</>
	);

	function hasAnyMedicalHistory(): boolean {
		return !!(
			patient.app?.trim() ||
			patient.apf?.trim() ||
			patient.apqx?.trim()
		);
	}
}

function formatGender(
	gender: string | undefined,
	textGet: (key: string) => string,
): string {
	switch (gender) {
		case "M":
			return textGet("clinical.patient_card.gender.male");
		case "F":
			return textGet("clinical.patient_card.gender.female");
		default:
			return textGet("clinical.patient_card.gender.other");
	}
}
