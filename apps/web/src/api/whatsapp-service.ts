import { createHttpService, type ServiceResponse } from "@pengi/shared";
import { apiWithTenant } from ".";
import type { PaginatedResponse } from "./clinical-service";

const whatsappService = createHttpService(apiWithTenant);

/** What the frontend needs to start Meta's Embedded Signup. */
export interface WhatsAppConfig {
	app_id: string;
	config_id: string;
	graph_version: string;
	embedded_signup_available: boolean;
}

export type WhatsAppTemplateStatus =
	| "APPROVED"
	| "PENDING"
	| "REJECTED"
	| "PAUSED"
	| "DISABLED"
	| "";

/** The tenant's connection. The access token is never sent. */
export interface WhatsAppAccount {
	connected: boolean;
	mode?: "manual" | "embedded";
	status?: "connected" | "token_invalid";
	waba_id?: string;
	phone_number_id?: string;
	display_phone?: string;
	verified_name?: string;
	template_name?: string;
	template_status: WhatsAppTemplateStatus;
	template_reason?: string;
	reminders_enabled: boolean;
	reminder_offsets: number[];
	connected_at?: string;
}

export type ConnectManualPayload = {
	waba_id: string;
	phone_number_id: string;
	access_token: string;
};

export type ConnectEmbeddedPayload = {
	code: string;
	waba_id: string;
	phone_number_id: string;
};

export type WhatsAppSettingsPayload = {
	reminders_enabled: boolean;
	reminder_offsets: number[];
};

export type WhatsAppMessageStatus =
	| "queued"
	| "sending"
	| "sent"
	| "delivered"
	| "read"
	| "replied"
	| "failed"
	| "skipped"
	| "received";

/**
 * reminder/test/reply/template/system are outbound; inbound is what the
 * patient wrote. template = a template sent from the inbox; system = an
 * automatic text (the consent question and its confirmation).
 */
export type WhatsAppMessageKind =
	| "reminder"
	| "test"
	| "reply"
	| "template"
	| "system"
	| "inbound";

export type WhatsAppContentType =
	| "text"
	| "button"
	| "image"
	| "audio"
	| "document"
	| "other";

/** One message: a row of the sent log or a bubble of a conversation. */
export interface WhatsAppMessage {
	id: number;
	created_at: string;
	kind: WhatsAppMessageKind;
	appointment_id: number | null;
	patient_id: number | null;
	patient_name: string;
	to_phone: string;
	template: string;
	offset_hours: number;
	status: WhatsAppMessageStatus;
	error_code: string;
	error_detail: string;
	reply: "" | "confirm" | "cancel";
	sent_at: string | null;
	delivered_at: string | null;
	read_at: string | null;
	replied_at: string | null;
	conversation_id: number | null;
	direction: "outbound" | "inbound";
	/** Text sent or received; for a button tap, the button label. */
	body: string;
	content_type: WhatsAppContentType | "";
	sent_by_user_id: number | null;
}

export interface WhatsAppMessageParams {
	page?: number;
	limit?: number;
	status?: WhatsAppMessageStatus;
	kind?: WhatsAppMessageKind;
	appointment_id?: number;
}

// ─── Conversations ───────────────────────────────────────────────────────────

/** One row of the inbox list. */
export interface WhatsAppConversation {
	id: number;
	created_at: string;
	phone: string;
	patient_id: number | null;
	patient_name: string;
	patient_whatsapp_opt_in: boolean;
	last_message_at: string | null;
	last_message_preview: string;
	last_direction: "outbound" | "inbound" | "";
	last_content_type: WhatsAppContentType | "";
	last_inbound_at: string | null;
	unread_count: number;
	window_open: boolean;
	/** End of the 24 h reply window; null if the patient never wrote. */
	window_expires_at: string | null;
}

export interface WhatsAppConversationPatient {
	id: number;
	first_name: string;
	last_name: string;
	document: string;
	phone: string;
	whatsapp_opt_in: boolean;
}

export interface WhatsAppConversationAppointment {
	id: number;
	title: string;
	date: string;
	start_time: string;
	end_time: string;
	status: string;
}

/** The conversation header. */
export interface WhatsAppConversationDetail extends WhatsAppConversation {
	patient: WhatsAppConversationPatient | null;
	next_appointment: WhatsAppConversationAppointment | null;
}

/** A page of a thread, oldest first; older pages use before=items[0].id. */
export interface WhatsAppThreadPage {
	items: WhatsAppMessage[];
	has_more: boolean;
}

export interface WhatsAppUnreadCount {
	/** Unread messages. */
	unread_count: number;
	/** Conversations with unread messages. */
	conversations: number;
}

export interface WhatsAppConversationParams {
	page?: number;
	limit?: number;
	search?: string;
	unread?: boolean;
}

// ─── Templates and usage ─────────────────────────────────────────────────────

/** A variable a template fills from the patient, the clinic or the appointment. */
export type WhatsAppTemplateVariable = "patient" | "clinic" | "date" | "time";

/** One entry of the template catalog with its approval status at Meta. */
export interface WhatsAppTemplate {
	name: string;
	/** Meta's status; "" when the template isn't created yet. */
	status: WhatsAppTemplateStatus;
	/** Meta's rejection reason, if any. */
	reason: string;
	variables: WhatsAppTemplateVariable[];
	/** Needs one of the patient's appointments (date and time come from it). */
	needs_appointment: boolean;
	/** Can be sent from the inbox (the reminder can't). */
	inbox: boolean;
	/** inbox && APPROVED. */
	usable: boolean;
	/** Body with {{1}}… placeholders. */
	body: string;
	/** Body filled with sample values. */
	preview: string;
	buttons: string[];
}

/** Templates sent this calendar month against the plan's cap. */
export interface WhatsAppUsage {
	used: number;
	/** -1 = unlimited. */
	limit: number;
	period_start: string;
	/** Exclusive. */
	period_end: string;
}

export type StartConversationPayload = {
	patient_id: number;
	template: string;
	appointment_id?: number;
};

export type SendTemplatePayload = {
	template: string;
	appointment_id?: number;
};

/** What sending a template returns: the conversation and the new message. */
export interface WhatsAppTemplateSent {
	conversation: WhatsAppConversation;
	message: WhatsAppMessage;
}

/** Longest free-text reply WhatsApp accepts (the backend enforces it too). */
export const MAX_REPLY_LENGTH = 4096;

/** Options of the calls that are also polled: polls fail silently. */
type PollOptions = { notifyError?: boolean };

// ─── Reminder offsets ────────────────────────────────────────────────────────

/** Hours before the appointment the user can pick from. */
export const REMINDER_OFFSET_CHOICES = [48, 24, 12, 2, 1] as const;
/** Most reminders per appointment (the backend enforces the same limit). */
export const MAX_REMINDER_OFFSETS = 3;

/**
 * Adds or removes `hours` from the selected offsets, sorted from the earliest
 * reminder to the latest. Adding past the limit leaves them unchanged.
 */
export function toggleReminderOffset(
	offsets: readonly number[],
	hours: number,
): number[] {
	if (offsets.includes(hours)) return offsets.filter((h) => h !== hours);
	if (offsets.length >= MAX_REMINDER_OFFSETS) return [...offsets];
	return [...offsets, hours].sort((a, b) => b - a);
}

/**
 * i18n key for a message's failure reason. Known codes (ours and Meta's) have
 * their own key; anything else falls back to the generic one.
 */
const KNOWN_ERROR_CODES = new Set([
	"invalid_phone",
	"no_phone",
	"not_connected",
	"token_unusable",
	"internal",
	"appointment_inactive",
	"monthly_limit",
	"131026",
	"131047",
	"131049",
	"132001",
	"190",
]);

export function messageErrorKey(code: string): string | null {
	if (!code) return null;
	return KNOWN_ERROR_CODES.has(code)
		? `whatsapp.message.error.${code}`
		: "whatsapp.message.error.unknown";
}

// ─── Calls ───────────────────────────────────────────────────────────────────

export const getWhatsAppConfig = async (): Promise<
	ServiceResponse<WhatsAppConfig>
> => whatsappService.get<WhatsAppConfig>("/whatsapp/config");

export const getWhatsAppAccount = async (): Promise<
	ServiceResponse<WhatsAppAccount>
> =>
	whatsappService.get<WhatsAppAccount>("/whatsapp/account", {
		notifyError: true,
	});

export const connectWhatsAppManual = async (
	payload: ConnectManualPayload,
): Promise<ServiceResponse<WhatsAppAccount>> =>
	whatsappService.post<WhatsAppAccount>("/whatsapp/connect/manual", payload, {
		notifySuccess: true,
		notifyError: true,
	});

export const connectWhatsAppEmbedded = async (
	payload: ConnectEmbeddedPayload,
): Promise<ServiceResponse<WhatsAppAccount>> =>
	whatsappService.post<WhatsAppAccount>("/whatsapp/connect/embedded", payload, {
		notifySuccess: true,
		notifyError: true,
	});

export const syncWhatsAppTemplate = async (): Promise<
	ServiceResponse<WhatsAppAccount>
> =>
	whatsappService.post<WhatsAppAccount>(
		"/whatsapp/template/sync",
		{},
		{ notifySuccess: true, notifyError: true },
	);

export const updateWhatsAppSettings = async (
	payload: WhatsAppSettingsPayload,
): Promise<ServiceResponse<WhatsAppAccount>> =>
	whatsappService.put<WhatsAppAccount>("/whatsapp/settings", payload, {
		notifySuccess: true,
		notifyError: true,
	});

export const disconnectWhatsApp = async (): Promise<ServiceResponse<null>> =>
	whatsappService.delete<null>("/whatsapp/account", {
		notifySuccess: true,
		notifyError: true,
	});

export const sendWhatsAppTest = async (
	phone: string,
): Promise<ServiceResponse<WhatsAppMessage>> =>
	whatsappService.post<WhatsAppMessage>(
		"/whatsapp/test",
		{ phone },
		{ notifySuccess: true, notifyError: true },
	);

export const getWhatsAppMessages = async (
	params: WhatsAppMessageParams = {},
): Promise<ServiceResponse<PaginatedResponse<WhatsAppMessage>>> => {
	const qs = new URLSearchParams();
	if (params.page) qs.set("page", String(params.page));
	if (params.limit) qs.set("limit", String(params.limit));
	if (params.status) qs.set("status", params.status);
	if (params.kind) qs.set("kind", params.kind);
	if (params.appointment_id)
		qs.set("appointment_id", String(params.appointment_id));
	const query = qs.toString();
	return whatsappService.get<PaginatedResponse<WhatsAppMessage>>(
		`/whatsapp/messages${query ? `?${query}` : ""}`,
		{ notifyError: true },
	);
};

export const getWhatsAppConversations = async (
	params: WhatsAppConversationParams = {},
	{ notifyError = false }: PollOptions = {},
): Promise<ServiceResponse<PaginatedResponse<WhatsAppConversation>>> => {
	const qs = new URLSearchParams();
	if (params.page) qs.set("page", String(params.page));
	if (params.limit) qs.set("limit", String(params.limit));
	if (params.search) qs.set("search", params.search);
	if (params.unread) qs.set("unread", "true");
	const query = qs.toString();
	return whatsappService.get<PaginatedResponse<WhatsAppConversation>>(
		`/whatsapp/conversations${query ? `?${query}` : ""}`,
		{ notifyError },
	);
};

export const getWhatsAppUnreadCount = async (): Promise<
	ServiceResponse<WhatsAppUnreadCount>
> =>
	whatsappService.get<WhatsAppUnreadCount>(
		"/whatsapp/conversations/unread-count",
	);

export const getWhatsAppConversation = async (
	id: number,
	{ notifyError = false }: PollOptions = {},
): Promise<ServiceResponse<WhatsAppConversationDetail>> =>
	whatsappService.get<WhatsAppConversationDetail>(
		`/whatsapp/conversations/${id}`,
		{ notifyError },
	);

export const getWhatsAppThread = async (
	id: number,
	{ before, limit = 50 }: { before?: number; limit?: number } = {},
	{ notifyError = false }: PollOptions = {},
): Promise<ServiceResponse<WhatsAppThreadPage>> => {
	const qs = new URLSearchParams({ limit: String(limit) });
	if (before) qs.set("before", String(before));
	return whatsappService.get<WhatsAppThreadPage>(
		`/whatsapp/conversations/${id}/messages?${qs.toString()}`,
		{ notifyError },
	);
};

/** Free-text reply inside the 24 h window. No success toast: it shows in the thread. */
export const sendWhatsAppReply = async (
	id: number,
	body: string,
): Promise<ServiceResponse<WhatsAppMessage>> =>
	whatsappService.post<WhatsAppMessage>(
		`/whatsapp/conversations/${id}/messages`,
		{ body },
		{ notifyError: true },
	);

export const markWhatsAppConversationRead = async (
	id: number,
): Promise<ServiceResponse<WhatsAppConversation>> =>
	whatsappService.post<WhatsAppConversation>(
		`/whatsapp/conversations/${id}/read`,
		{},
	);

export const linkWhatsAppConversationPatient = async (
	id: number,
	patientId: number,
): Promise<ServiceResponse<WhatsAppConversationDetail>> =>
	whatsappService.put<WhatsAppConversationDetail>(
		`/whatsapp/conversations/${id}/patient`,
		{ patient_id: patientId },
		{ notifySuccess: true, notifyError: true },
	);

export const getWhatsAppTemplates = async (): Promise<
	ServiceResponse<WhatsAppTemplate[]>
> =>
	whatsappService.get<WhatsAppTemplate[]>("/whatsapp/templates", {
		notifyError: true,
	});

/** Silent: the usage is a side note wherever it shows. */
export const getWhatsAppUsage = async (): Promise<
	ServiceResponse<WhatsAppUsage>
> => whatsappService.get<WhatsAppUsage>("/whatsapp/usage");

/** Writes first to a patient with a template (creates the conversation if needed). */
export const startWhatsAppConversation = async (
	payload: StartConversationPayload,
): Promise<ServiceResponse<WhatsAppTemplateSent>> =>
	whatsappService.post<WhatsAppTemplateSent>(
		"/whatsapp/conversations/start",
		payload,
		{ notifySuccess: true, notifyError: true },
	);

/**
 * Sends a template in an existing conversation (also with the reply window
 * closed). No success toast: it shows in the thread.
 */
export const sendWhatsAppTemplate = async (
	conversationId: number,
	payload: SendTemplatePayload,
): Promise<ServiceResponse<WhatsAppTemplateSent>> =>
	whatsappService.post<WhatsAppTemplateSent>(
		`/whatsapp/conversations/${conversationId}/template`,
		payload,
		{ notifyError: true },
	);
