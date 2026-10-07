import { parseDateOnly, useText } from "@pengi/shared";
import {
	Button,
	cn,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
	Skeleton,
	Text,
} from "@pengi/ui";
import { AlertTriangle, BellOff, Gauge, Loader2, Send } from "lucide-react";
import React from "react";
import {
	type Appointment,
	getPatientAppointments,
	type Patient,
} from "@/api/clinical-service";
import {
	getWhatsAppTemplates,
	getWhatsAppUsage,
	sendWhatsAppTemplate,
	startWhatsAppConversation,
	type WhatsAppTemplate,
	type WhatsAppTemplateSent,
	type WhatsAppUsage,
} from "@/api/whatsapp-service";
import {
	inboxTemplates,
	templateNeedsAppointment,
	templatePreview,
	usageSummary,
} from "@/lib/whatsapp-templates";
import { PatientSearch, patientName } from "./patient-search";

/** The recipient of a template: what the dialog needs to know about them. */
export interface TemplateRecipient {
	id: number;
	name: string;
	phone: string;
	whatsapp_opt_in: boolean;
}

export function recipientFromPatient(p: Patient): TemplateRecipient {
	return {
		id: p.ID,
		name: patientName(p),
		phone: p.phone,
		whatsapp_opt_in: !!p.whatsapp_opt_in,
	};
}

interface TemplateMessageDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	/**
	 * Sends in this conversation (its patient is fixed, the window may be
	 * closed). Without it the dialog starts a conversation: "Nuevo mensaje".
	 */
	conversationId?: number;
	/**
	 * The recipient: preselected for a new conversation (it can still be
	 * changed), the linked patient of an existing one (null = unlinked).
	 */
	patient?: TemplateRecipient | null;
	onSent: (result: WhatsAppTemplateSent) => void;
}

/**
 * Sends one of the inbox templates (approved by Meta): pick the patient (new
 * conversation only), the template with its preview and, when it needs one,
 * the appointment. Each template counts against the plan's monthly cap.
 */
export function TemplateMessageDialog({
	open,
	onOpenChange,
	conversationId,
	patient: initialPatient = null,
	onSent,
}: TemplateMessageDialogProps) {
	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-lg">
				{/* Mounted only while open, so closing resets every choice. */}
				{open && (
					<TemplateMessageForm
						conversationId={conversationId}
						initialPatient={initialPatient}
						onSent={(result) => {
							onSent(result);
							onOpenChange(false);
						}}
					/>
				)}
			</DialogContent>
		</Dialog>
	);
}

function TemplateMessageForm({
	conversationId,
	initialPatient,
	onSent,
}: {
	conversationId?: number;
	initialPatient: TemplateRecipient | null;
	onSent: (result: WhatsAppTemplateSent) => void;
}) {
	const { textGet, formatDate } = useText();
	const inConversation = conversationId !== undefined;
	const [patient, setPatient] = React.useState<TemplateRecipient | null>(
		initialPatient,
	);
	const [templates, setTemplates] = React.useState<WhatsAppTemplate[] | null>(
		null,
	);
	const [usage, setUsage] = React.useState<WhatsAppUsage | null>(null);
	const [templateName, setTemplateName] = React.useState("");
	// Appointments with the patient they belong to (stale while it changes).
	const [appointments, setAppointments] = React.useState<{
		patientId: number;
		items: Appointment[];
	} | null>(null);
	const [appointmentId, setAppointmentId] = React.useState("");
	const [sending, setSending] = React.useState(false);

	React.useEffect(() => {
		let cancelled = false;
		Promise.all([getWhatsAppTemplates(), getWhatsAppUsage()]).then(([t, u]) => {
			if (cancelled) return;
			const list = t.success ? inboxTemplates(t.data ?? []) : [];
			setTemplates(list);
			setTemplateName(list.find((x) => x.usable)?.name ?? "");
			if (u.success) setUsage(u.data);
		});
		return () => {
			cancelled = true;
		};
	}, []);

	const template = templates?.find((t) => t.name === templateName) ?? null;
	const needsAppointment = !!template && templateNeedsAppointment(template);
	const patientId = patient?.id ?? null;

	React.useEffect(() => {
		if (!needsAppointment || patientId === null) return;
		let cancelled = false;
		getPatientAppointments(patientId).then((res) => {
			if (cancelled) return;
			setAppointments({
				patientId,
				items: res.success ? (res.data ?? []) : [],
			});
		});
		return () => {
			cancelled = true;
		};
	}, [needsAppointment, patientId]);

	const patientAppointments =
		appointments && appointments.patientId === patientId
			? appointments.items
			: null;
	const usageInfo = usage ? usageSummary(usage) : null;
	const noConsent = !!patient && !patient.whatsapp_opt_in;
	const canSend =
		!!patient &&
		!noConsent &&
		!!template?.usable &&
		(!needsAppointment || appointmentId !== "") &&
		!usageInfo?.reached &&
		!sending;

	function pickPatient(p: Patient) {
		setPatient(recipientFromPatient(p));
		setAppointmentId("");
	}

	function appointmentLabel(a: Appointment) {
		return `${formatDate(parseDateOnly(a.date), "medium")} · ${a.start_time}`;
	}

	async function send() {
		if (!canSend || !patient || !template) return;
		setSending(true);
		const appointment = needsAppointment ? Number(appointmentId) : undefined;
		const res = inConversation
			? await sendWhatsAppTemplate(conversationId, {
					template: template.name,
					appointment_id: appointment,
				})
			: await startWhatsAppConversation({
					patient_id: patient.id,
					template: template.name,
					appointment_id: appointment,
				});
		setSending(false);
		if (res.success) onSent(res.data);
	}

	return (
		<>
			<DialogHeader>
				<DialogTitle>
					<Text
						uuid={
							inConversation
								? "whatsapp.send_template.title"
								: "whatsapp.new_message.title"
						}
					/>
				</DialogTitle>
				<DialogDescription>
					<Text
						uuid={
							inConversation
								? "whatsapp.send_template.description"
								: "whatsapp.new_message.description"
						}
					/>
				</DialogDescription>
			</DialogHeader>

			<div className="grid gap-4">
				{/* Recipient */}
				{patient ? (
					<div className="grid gap-2">
						<div className="flex items-center justify-between gap-2 rounded-md border px-3 py-2">
							<span className="grid min-w-0">
								<span className="truncate text-sm font-medium">
									{patient.name}
								</span>
								<span className="text-xs text-muted-foreground tabular-nums">
									{patient.phone}
								</span>
							</span>
							{!inConversation && (
								<Button
									variant="ghost"
									size="sm"
									onClick={() => setPatient(null)}
								>
									<Text uuid="whatsapp.new_message.change_patient" />
								</Button>
							)}
						</div>
						{noConsent && (
							<p className="flex items-start gap-1.5 text-xs text-amber-700 dark:text-amber-400">
								<BellOff className="mt-0.5 h-3.5 w-3.5 shrink-0" />
								<Text uuid="whatsapp.new_message.no_opt_in" />
							</p>
						)}
					</div>
				) : inConversation ? (
					<p className="flex items-start gap-1.5 text-sm text-amber-700 dark:text-amber-400">
						<AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
						<Text uuid="whatsapp.send_template.unlinked" />
					</p>
				) : (
					<div className="grid gap-1">
						<PatientSearch
							placeholderKey="whatsapp.new_message.search"
							onPick={pickPatient}
							filter={(p) => !!p.phone}
							showOptIn
						/>
						<p className="text-xs text-muted-foreground">
							<Text uuid="whatsapp.new_message.phone_only" />
						</p>
					</div>
				)}

				{/* Template */}
				{patient && (
					<div className="grid gap-2">
						<p className="text-sm font-medium">
							<Text uuid="whatsapp.new_message.template" />
						</p>
						{templates === null ? (
							<Skeleton className="h-20" />
						) : templates.length === 0 ? (
							<p className="text-sm text-muted-foreground">
								<Text uuid="whatsapp.new_message.no_templates" />
							</p>
						) : (
							<div className="flex flex-wrap gap-2">
								{templates.map((t) => (
									<button
										key={t.name}
										type="button"
										aria-pressed={t.name === templateName}
										disabled={!t.usable}
										onClick={() => {
											setTemplateName(t.name);
											setAppointmentId("");
										}}
										title={
											t.usable
												? undefined
												: textGet(
														`settings.whatsapp.template.status.${t.status || "none"}`,
													)
										}
										className={cn(
											"rounded-full border px-3 py-1 text-xs font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50",
											t.name === templateName
												? "border-primary bg-primary text-primary-foreground"
												: "bg-background hover:bg-muted",
										)}
									>
										<Text uuid={`whatsapp.template.name.${t.name}`} />
									</button>
								))}
							</div>
						)}
						{templates &&
							templates.length > 0 &&
							!templates.some((t) => t.usable) && (
								<p className="text-xs text-muted-foreground">
									<Text uuid="whatsapp.new_message.none_approved" />
								</p>
							)}
						{template && (
							<div className="grid gap-1">
								<p className="text-xs text-muted-foreground">
									<Text uuid="whatsapp.new_message.preview" />
								</p>
								<p className="whitespace-pre-wrap break-words rounded-2xl rounded-br-sm bg-emerald-100 px-3 py-2 text-sm text-emerald-950 dark:bg-emerald-900/50 dark:text-emerald-50">
									{templatePreview(template)}
								</p>
								<p className="text-xs text-muted-foreground">
									<Text uuid="whatsapp.new_message.preview_hint" />
								</p>
							</div>
						)}
					</div>
				)}

				{/* Appointment */}
				{patient && needsAppointment && (
					<div className="grid gap-2">
						<p className="text-sm font-medium">
							<Text uuid="whatsapp.new_message.appointment" />
						</p>
						{patientAppointments === null ? (
							<Skeleton className="h-9" />
						) : patientAppointments.length === 0 ? (
							<p className="text-sm text-muted-foreground">
								<Text uuid="whatsapp.new_message.appointment.none" />
							</p>
						) : (
							<Select
								value={appointmentId}
								onValueChange={(v) => setAppointmentId(String(v ?? ""))}
							>
								<SelectTrigger className="w-full">
									<SelectValue>
										{(() => {
											const a = patientAppointments.find(
												(x) => String(x.ID) === appointmentId,
											);
											return a
												? appointmentLabel(a)
												: textGet(
														"whatsapp.new_message.appointment.placeholder",
													);
										})()}
									</SelectValue>
								</SelectTrigger>
								<SelectContent>
									{patientAppointments.map((a) => (
										<SelectItem key={a.ID} value={String(a.ID)}>
											{appointmentLabel(a)}
											{a.title ? ` · ${a.title}` : ""}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
						)}
					</div>
				)}

				{/* Monthly quota */}
				{patient && usage && usageInfo && (
					<p
						className={cn(
							"flex items-start gap-1.5 text-xs",
							usageInfo.reached
								? "text-destructive"
								: usageInfo.warning
									? "text-amber-700 dark:text-amber-400"
									: "text-muted-foreground",
						)}
					>
						<Gauge className="mt-0.5 h-3.5 w-3.5 shrink-0" />
						<Text
							uuid={
								usageInfo.reached
									? "whatsapp.new_message.usage.reached"
									: usageInfo.unlimited
										? "whatsapp.new_message.usage.unlimited"
										: "whatsapp.new_message.usage"
							}
							values={{ used: usage.used, limit: usage.limit }}
						/>
					</p>
				)}
			</div>

			<DialogFooter>
				<Button onClick={send} disabled={!canSend}>
					{sending ? <Loader2 className="animate-spin" /> : <Send />}
					<Text uuid="whatsapp.new_message.send" />
				</Button>
			</DialogFooter>
		</>
	);
}
