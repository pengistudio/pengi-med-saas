import { useMessageStore, useText } from "@pengi/shared";
import {
	Button,
	Calendar,
	Form,
	FormTextArea,
	Popover,
	PopoverContent,
	PopoverTrigger,
	Text,
} from "@pengi/ui";
import { addDays, differenceInCalendarDays, format } from "date-fns";
import { enUS, es } from "date-fns/locale";
import { CalendarIcon, Loader2, Minus, Plus } from "lucide-react";
import React from "react";
import { type UseFormReturn, useWatch } from "react-hook-form";
import { useSearchParams } from "react-router";
import { z } from "zod";
import {
	createMedicalCertificate,
	downloadMedicalCertificatePdf,
	emailMedicalCertificate,
	getPatientById,
	type Patient,
} from "@/api/clinical-service";
import {
	type DocumentSignature,
	signMedicalCertificate,
} from "@/api/signature-service";
import { DocumentPageHeader } from "@/components/features/medical-documents/document-page-header";
import {
	DocumentWorkflow,
	type DocumentWorkflowProps,
	useSavedBaseline,
} from "@/components/features/medical-documents/document-workflow";
import { cn } from "@/lib/utils";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const MAX_REST_DAYS = 365;

const certificateSchema = z.object({
	diagnosis: z
		.string()
		.trim()
		.min(1, "dialog.medical_certificate.diagnosis.required"),
	observations: z.string(),
	rest_days: z.number().int().min(0).max(MAX_REST_DAYS),
	rest_from: z.date().optional(),
	rest_to: z.date().optional(),
});

type CertificateFormValues = z.input<typeof certificateSchema>;

function DatePicker({
	id,
	value,
	onChange,
	disabled,
	fromDate,
}: {
	id: string;
	value?: Date;
	onChange: (date: Date | undefined) => void;
	disabled?: boolean;
	fromDate?: Date;
}) {
	const { textGet } = useText();
	const lang = useMessageStore((state) => state.lang);
	const locale = lang === "es" ? es : enUS;

	return (
		<Popover>
			<PopoverTrigger
				render={
					<Button
						id={id}
						type="button"
						variant="outline"
						disabled={disabled}
						className={cn(
							"h-10 w-full justify-start bg-background text-left font-normal",
							!value && "text-muted-foreground",
						)}
					/>
				}
			>
				<CalendarIcon className="mr-2 size-4" />
				{value
					? format(value, "PP", { locale })
					: textGet("form.calendar.pick_date")}
			</PopoverTrigger>
			<PopoverContent className="w-auto p-0" align="start">
				<Calendar
					mode="single"
					selected={value}
					onSelect={onChange}
					locale={locale}
					disabled={fromDate ? { before: fromDate } : undefined}
					defaultMonth={value ?? fromDate}
				/>
			</PopoverContent>
		</Popover>
	);
}

/** "lunes 27 de septiembre [de 2026]" / "Monday, September 27[, 2026]". */
function formatRestDate(
	date: Date,
	lang: string | undefined,
	withYear: boolean,
) {
	if (lang === "es") {
		return format(
			date,
			withYear ? "EEEE d 'de' MMMM 'de' yyyy" : "EEEE d 'de' MMMM",
			{ locale: es },
		);
	}
	return format(date, withYear ? "EEEE, MMMM d, yyyy" : "EEEE, MMMM d", {
		locale: enUS,
	});
}

/**
 * Rest days and the rest period kept in agreement: the count is inclusive,
 * so 3 days from the 27th ends on the 29th. Editing any of the three
 * recalculates the others.
 */
function RestPeriod({
	field,
	readOnly,
}: {
	field: UseFormReturn<CertificateFormValues>;
	readOnly: boolean;
}) {
	const { textGet } = useText();
	const lang = useMessageStore((state) => state.lang);
	const [days, from, to] = useWatch({
		control: field.control,
		name: ["rest_days", "rest_from", "rest_to"],
	});

	const set = (values: Partial<CertificateFormValues>) => {
		for (const [key, value] of Object.entries(values)) {
			field.setValue(key as keyof CertificateFormValues, value as never, {
				shouldDirty: true,
			});
		}
	};

	function setDays(next: number) {
		const clamped = Math.min(MAX_REST_DAYS, Math.max(0, Math.round(next) || 0));
		if (clamped === 0) {
			set({ rest_days: 0, rest_from: undefined, rest_to: undefined });
			return;
		}
		const start = from ?? new Date();
		set({
			rest_days: clamped,
			rest_from: start,
			rest_to: addDays(start, clamped - 1),
		});
	}

	function setFrom(date: Date | undefined) {
		if (!date) return;
		const length = days > 0 ? days : 1;
		set({
			rest_days: length,
			rest_from: date,
			rest_to: addDays(date, length - 1),
		});
	}

	function setTo(date: Date | undefined) {
		if (!date) return;
		const start = from && date >= from ? from : date;
		set({
			rest_from: start,
			rest_to: date,
			rest_days: differenceInCalendarDays(date, start) + 1,
		});
	}

	return (
		<section className="space-y-4">
			<h2 className="text-base font-semibold">
				<Text uuid="dialog.medical_certificate.rest.title" />
			</h2>

			<div className="grid gap-5 rounded-xl border bg-muted/30 p-5 sm:grid-cols-[auto_minmax(0,1fr)_minmax(0,1fr)] sm:items-end">
				<div className="space-y-1.5">
					<label htmlFor="rest_days" className="text-xs text-muted-foreground">
						<Text uuid="dialog.medical_certificate.rest_days" />
					</label>
					<div className="flex items-center gap-1">
						<Button
							type="button"
							variant="outline"
							size="icon"
							disabled={readOnly || days <= 0}
							onClick={() => setDays(days - 1)}
							aria-label={textGet("dialog.medical_certificate.rest.decrease")}
						>
							<Minus className="size-4" />
						</Button>
						<input
							id="rest_days"
							type="number"
							inputMode="numeric"
							min={0}
							max={MAX_REST_DAYS}
							value={days}
							readOnly={readOnly}
							onChange={(e) => setDays(Number(e.target.value))}
							className="h-10 w-16 rounded-md bg-transparent text-center text-3xl font-semibold tabular-nums outline-none focus-visible:ring-3 focus-visible:ring-ring/50 [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
						/>
						<Button
							type="button"
							variant="outline"
							size="icon"
							disabled={readOnly || days >= MAX_REST_DAYS}
							onClick={() => setDays(days + 1)}
							aria-label={textGet("dialog.medical_certificate.rest.increase")}
						>
							<Plus className="size-4" />
						</Button>
					</div>
				</div>

				<div className="space-y-1.5">
					<label htmlFor="rest_from" className="text-xs text-muted-foreground">
						<Text uuid="dialog.medical_certificate.rest.from_label" />
					</label>
					<DatePicker
						id="rest_from"
						value={from}
						onChange={setFrom}
						disabled={readOnly}
					/>
				</div>

				<div className="space-y-1.5">
					<label htmlFor="rest_to" className="text-xs text-muted-foreground">
						<Text uuid="dialog.medical_certificate.rest.to_label" />
					</label>
					<DatePicker
						id="rest_to"
						value={to}
						onChange={setTo}
						disabled={readOnly}
						fromDate={from}
					/>
				</div>

				<p
					aria-live="polite"
					className={cn(
						"text-sm sm:col-span-3",
						days > 0 ? "font-medium" : "text-muted-foreground",
					)}
				>
					{days > 0 && from && to ? (
						<>
							{days}{" "}
							{textGet(
								days === 1
									? "dialog.medical_certificate.rest.day"
									: "dialog.medical_certificate.rest.days",
							)}
							, {textGet("dialog.medical_certificate.rest.from_word")}{" "}
							{formatRestDate(
								from,
								lang,
								from.getFullYear() !== to.getFullYear(),
							)}{" "}
							{textGet("dialog.medical_certificate.rest.to_word")}{" "}
							{formatRestDate(to, lang, true)}.
						</>
					) : (
						<Text uuid="dialog.medical_certificate.rest.none" />
					)}
				</p>
			</div>
		</section>
	);
}

interface CertificateEditorProps
	extends Omit<DocumentWorkflowProps, "isDirty" | "saveLabel"> {
	field: UseFormReturn<CertificateFormValues>;
	savedVersion: number;
}

function CertificateEditor({
	field,
	savedVersion,
	...workflow
}: CertificateEditorProps) {
	useSavedBaseline(field, savedVersion);
	const readOnly = Boolean(workflow.signature);

	return (
		<div className="grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_18rem]">
			<div className="space-y-8">
				<section className="space-y-2">
					<label htmlFor="diagnosis" className="text-base font-semibold">
						<Text uuid="dialog.medical_certificate.diagnosis" />
					</label>
					<FormTextArea
						field={field}
						name="diagnosis"
						readOnly={readOnly}
						className="min-h-20"
					/>
				</section>

				<RestPeriod field={field} readOnly={readOnly} />

				<section className="space-y-2">
					<label htmlFor="observations" className="text-base font-semibold">
						<Text uuid="dialog.medical_certificate.observations" />{" "}
						<span className="text-sm font-normal text-muted-foreground">
							<Text uuid="form.optional" />
						</span>
					</label>
					<FormTextArea
						field={field}
						name="observations"
						readOnly={readOnly}
						className="min-h-20"
					/>
				</section>

				{readOnly && (
					<p className="text-sm text-muted-foreground">
						<Text uuid="medical_document.signed_readonly" />
					</p>
				)}
			</div>

			<DocumentWorkflow
				{...workflow}
				saveLabel={<Text uuid="dialog.medical_certificate.save_certificate" />}
				isDirty={field.formState.isDirty}
			/>
		</div>
	);
}

export default function GenerateMedicalCertificatePage() {
	const [searchParams] = useSearchParams();
	const patientId = Number(searchParams.get("patient_id"));

	const [patient, setPatient] = React.useState<Patient | null>(null);
	const [loadingPatient, setLoadingPatient] = React.useState(true);
	const [saving, setSaving] = React.useState(false);
	const [printing, setPrinting] = React.useState(false);
	const [savedCertificateId, setSavedCertificateId] = React.useState<
		number | null
	>(null);
	const [savedVersion, setSavedVersion] = React.useState(0);
	const [signature, setSignature] = React.useState<DocumentSignature | null>(
		null,
	);

	React.useEffect(() => {
		if (!patientId) return;
		getPatientById(patientId).then((res) => {
			if (res.success && res.data) setPatient(res.data);
			setLoadingPatient(false);
		});
	}, [patientId]);

	async function onSubmit(values: z.infer<typeof certificateSchema>) {
		setSaving(true);
		const hasRest = values.rest_days > 0;
		const res = await createMedicalCertificate(patientId, {
			diagnosis: values.diagnosis,
			observations: values.observations,
			rest_days: hasRest ? values.rest_days : null,
			rest_from:
				hasRest && values.rest_from ? values.rest_from.toISOString() : null,
			rest_to: hasRest && values.rest_to ? values.rest_to.toISOString() : null,
		});
		if (res.success && res.data) {
			setSavedCertificateId(res.data.ID);
			setSavedVersion((v) => v + 1);
			setSignature(null);
		}
		setSaving(false);
	}

	async function handleSign() {
		if (!savedCertificateId) return false;
		const res = await signMedicalCertificate(savedCertificateId);
		if (res.success && res.data) setSignature(res.data);
		return res.success;
	}

	async function handlePrint() {
		if (!savedCertificateId) return;
		setPrinting(true);
		const res = await downloadMedicalCertificatePdf(savedCertificateId);
		if (res.success && res.data) {
			const url = window.URL.createObjectURL(res.data);
			window.open(url, "_blank");
		}
		setPrinting(false);
	}

	async function handleSendEmail(email: string) {
		if (!savedCertificateId || !email) return;
		await emailMedicalCertificate(savedCertificateId, email);
	}

	return (
		<DashboardLayout>
			<div className="mx-auto w-full max-w-6xl space-y-6">
				<DocumentPageHeader
					patientId={patientId}
					patient={patient}
					title="dialog.medical_certificate.title"
					description="dialog.medical_certificate.description"
				/>

				{loadingPatient || !patient ? (
					<div className="flex justify-center py-16">
						<Loader2 className="size-6 animate-spin text-muted-foreground" />
					</div>
				) : (
					<Form
						schema={certificateSchema}
						defaultValues={{
							diagnosis: patient.diagnosis || "",
							observations: "",
							rest_days: 0,
						}}
						onSubmit={onSubmit}
					>
						{(field) => (
							<CertificateEditor
								field={field}
								savedVersion={savedVersion}
								saving={saving}
								printing={printing}
								isSaved={savedCertificateId !== null}
								signature={signature}
								patientEmail={patient.email ?? ""}
								onSign={handleSign}
								onPrint={handlePrint}
								onSendEmail={handleSendEmail}
							/>
						)}
					</Form>
				)}
			</div>
		</DashboardLayout>
	);
}
