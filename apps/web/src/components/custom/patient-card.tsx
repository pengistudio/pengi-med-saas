import {
	AlertCircle,
	AlertTriangle,
	FileText,
	Pencil,
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Text } from "@/components/ui/text";
import useTenantSettings from "@/hooks/use-tenant-settings";
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

interface PatientCardProps {
	patient: PatientCardData;
	headerAction?: React.ReactNode;
	onEditPatient?: () => void;
}

const NotAvailable = () => <span className="text-muted-foreground">—</span>;

export default function PatientCard({
	patient,
	headerAction,
	onEditPatient,
}: PatientCardProps) {
	const [showPrescription, setShowPrescription] = React.useState(false);
	const { textGet } = useText();
	const { settings } = useTenantSettings();
	const useAgeInput = settings.clinical.patient_age_input;
	const age = patient.birth_date
		? Math.floor(
				(Date.now() - new Date(patient.birth_date).getTime()) /
					(365.25 * 24 * 60 * 60 * 1000),
			)
		: null;

	const displayName =
		patient.full_name || `${patient.last_name} ${patient.first_name}`;

	const allergies = parseAllergies(patient.allergies);
	const hasMedicalHistory = !!(
		patient.app?.trim() ||
		patient.apf?.trim() ||
		patient.apqx?.trim()
	);

	return (
		<>
			<Card className="h-fit border-l-4 border-l-primary">
				<CardHeader className="space-y-4">
					<div className="flex items-start justify-between gap-4">
						<div className="space-y-2 min-w-0">
							<CardTitle className="text-2xl leading-tight wrap-break-word max-w-[250px]">
								{displayName}
							</CardTitle>
							<CardDescription className="flex items-center gap-2">
								<FileText className="h-4 w-4 shrink-0" />
								<Text uuid="clinical.patient_card.document" />{" "}
								<span className="text-foreground">{patient.document}</span>
							</CardDescription>
						</div>
						<div className="flex flex-col items-end gap-2 shrink-0">
							{patient.critical && (
								<Badge
									variant="destructive"
									className="px-4 py-1 shrink-0 hover:cursor-default"
								>
									<AlertCircle className="h-4 w-4 mr-1" />
									<Text uuid="clinical.patient_card.critical" />
								</Badge>
							)}
							{allergies.length > 0 && (
								<Badge className="px-4 py-1 shrink-0 hover:cursor-default bg-amber-500 hover:bg-amber-500 text-white">
									<AlertTriangle className="h-4 w-4 mr-1" />
									<Text uuid="clinical.patient_card.has_allergies" />
								</Badge>
							)}
							{headerAction}
						</div>
					</div>
				</CardHeader>

				<CardContent className="pb-4">
					<Tabs defaultValue="resumen">
						<TabsList className="w-full mb-4">
							<TabsTrigger value="resumen" className="flex-1">
								<Text uuid="clinical.patient_card.tab.summary" />
							</TabsTrigger>
							<TabsTrigger value="antecedentes" className="flex-1">
								<Text uuid="clinical.patient_card.tab.history" />
							</TabsTrigger>
							<TabsTrigger value="notas" className="flex-1">
								<Text uuid="clinical.patient_card.tab.notes" />
							</TabsTrigger>
						</TabsList>

						{/* ── Tab: Resumen ── */}
						<TabsContent value="resumen" className="space-y-5 mt-0">
							{/* Personal */}
							<div className="space-y-3">
								<h3 className="font-semibold flex items-center gap-2 text-xs text-muted-foreground uppercase tracking-wide">
									<UserRound className="h-3.5 w-3.5" />
									<Text uuid="clinical.patient_card.personal_info" />
								</h3>
								<div className="grid gap-2.5 text-sm">
									<div className="grid grid-cols-[110px_1fr] items-center">
										<span className="text-muted-foreground">
											{useAgeInput ? (
												<Text uuid="clinical.patient_card.age" />
											) : (
												<Text uuid="clinical.patient_card.birth_date" />
											)}
										</span>
										<span>
											{useAgeInput ? (
												age !== null ? (
													`${age} años`
												) : (
													<NotAvailable />
												)
											) : (
												dateParser(patient.birth_date, { dateStyle: "medium" })
											)}
										</span>
									</div>
									<div className="grid grid-cols-[110px_1fr] items-center">
										<span className="text-muted-foreground">
											<Text uuid="clinical.patient_card.gender" />
										</span>
										<span>{formatGender(patient.gender, textGet)}</span>
									</div>
									<div className="grid grid-cols-[110px_1fr] items-center">
										<span className="text-muted-foreground">
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
										<div className="grid grid-cols-[110px_1fr] items-center">
											<span className="text-muted-foreground">
												<Text uuid="clinical.patient_card.family_phone" />
											</span>
											<span>{patient.family_number}</span>
										</div>
									)}
									{patient.second_family_number && (
										<div className="grid grid-cols-[110px_1fr] items-center">
											<span className="text-muted-foreground">
												<Text uuid="clinical.patient_card.family_phone_2" />
											</span>
											<span>{patient.second_family_number}</span>
										</div>
									)}
								</div>
							</div>

							<Separator />

							{/* Médica */}
							<div className="space-y-3">
								<h3 className="font-semibold flex items-center gap-2 text-xs text-muted-foreground uppercase tracking-wide">
									<Stethoscope className="h-3.5 w-3.5" />
									<Text uuid="clinical.patient_card.medical_info" />
								</h3>
								<div className="grid gap-2.5 text-sm">
									<div className="grid grid-cols-[110px_1fr] items-center">
										<span className="text-muted-foreground">
											<Text uuid="clinical.patient_card.insurance" />
										</span>
										{patient.insurance ? (
											<span>{patient.insurance}</span>
										) : (
											<NotAvailable />
										)}
									</div>
									<div className="grid grid-cols-[110px_1fr] items-center">
										<span className="text-muted-foreground">
											<Text uuid="clinical.patient_card.medic" />
										</span>
										{patient.medic ? (
											<span>{patient.medic}</span>
										) : (
											<NotAvailable />
										)}
									</div>
									<div className="grid grid-cols-[110px_1fr] items-start">
										<span className="text-muted-foreground">
											<Text uuid="clinical.patient_card.diagnosis" />
										</span>
										{patient.diagnosis ? (
											<span className="wrap-break-word">
												{patient.diagnosis}
											</span>
										) : (
											<NotAvailable />
										)}
									</div>
								</div>
							</div>

							{/* Alergias */}
							{allergies.length > 0 && (
								<>
									<Separator />
									<div className="space-y-2.5">
										<h3 className="font-semibold flex items-center gap-2 text-xs text-amber-600 uppercase tracking-wide">
											<AlertTriangle className="h-3.5 w-3.5" />
											<Text uuid="clinical.patient_card.allergies" />
										</h3>
										<div className="flex flex-wrap gap-2">
											{allergies.map((allergy) => (
												<Badge
													key={allergy}
													className="bg-amber-100 text-amber-800 border-amber-300 hover:bg-amber-100"
													variant="outline"
												>
													{allergy}
												</Badge>
											))}
										</div>
									</div>
								</>
							)}

							{/* Receta */}
							{patient.latest_prescription && (
								<>
									<Separator />
									<div className="space-y-3">
										<h3 className="font-semibold flex items-center gap-2 text-xs text-muted-foreground uppercase tracking-wide">
											<Pill className="h-3.5 w-3.5" />
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
						</TabsContent>

						{/* ── Tab: Antecedentes ── */}
						<TabsContent value="antecedentes" className="mt-0">
							{hasMedicalHistory ? (
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
							) : (
								<p className="text-sm text-muted-foreground text-center py-8">
									<Text uuid="clinical.patient_card.no_history" />
								</p>
							)}
						</TabsContent>

						{/* ── Tab: Notas ── */}
						<TabsContent value="notas" className="mt-0">
							{patient.notes ? (
								<div className="space-y-3">
									<p className="text-sm whitespace-pre-wrap">{patient.notes}</p>
									{onEditPatient && (
										<Button
											variant="ghost"
											size="sm"
											onClick={onEditPatient}
											className="text-muted-foreground"
										>
											<Pencil className="h-3.5 w-3.5 mr-1.5" />
											<Text uuid="clinical.patient_card.edit_notes" />
										</Button>
									)}
								</div>
							) : (
								<div className="flex flex-col items-center gap-3 py-8">
									<p className="text-sm text-muted-foreground">
										<Text uuid="clinical.patient_card.no_notes" />
									</p>
									{onEditPatient && (
										<Button variant="outline" size="sm" onClick={onEditPatient}>
											<Pencil className="h-3.5 w-3.5 mr-1.5" />
											<Text uuid="clinical.patient_card.add_notes" />
										</Button>
									)}
								</div>
							)}
						</TabsContent>
					</Tabs>
				</CardContent>
			</Card>

			<PrescriptionDialog
				open={showPrescription}
				onOpenChange={setShowPrescription}
				prescription={patient.latest_prescription}
			/>
		</>
	);
}

function parseAllergies(allergies?: string): string[] {
	if (!allergies?.trim()) return [];
	return allergies
		.split(",")
		.map((a) => a.trim())
		.filter(Boolean);
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
