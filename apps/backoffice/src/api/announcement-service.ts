import { createHttpService, type ServiceResponse } from "@pengi/shared";
import { resource } from "@/lib/resource/http-resource";
import { api } from ".";

const httpService = createHttpService(api);

export const ANNOUNCEMENT_SCOPES = ["global", "company", "user"] as const;
export const ANNOUNCEMENT_LEVELS = [
	"info",
	"success",
	"warning",
	"critical",
] as const;

export type AnnouncementScope = (typeof ANNOUNCEMENT_SCOPES)[number];
export type AnnouncementLevel = (typeof ANNOUNCEMENT_LEVELS)[number];
export type AnnouncementStatus = "scheduled" | "sent" | "cancelled" | "failed";

export interface Announcement {
	ID: number;
	CreatedAt: string;
	UpdatedAt: string;
	scope: AnnouncementScope;
	company_id: number | null;
	company_name: string;
	user_id: number | null;
	user_name: string;
	title: string;
	body: string;
	level: AnnouncementLevel;
	action_url: string;
	scheduled_at: string | null;
	sent_at: string | null;
	status: AnnouncementStatus;
	recipient_count: number;
}

export interface CreateAnnouncementRequest extends Record<string, unknown> {
	scope: AnnouncementScope;
	company_id?: number;
	user_id?: number;
	title: string;
	body: string;
	level: AnnouncementLevel;
	action_url?: string;
	/** ISO date; omitted to send right away. */
	scheduled_at?: string;
}

export const announcements = resource<
	Announcement,
	CreateAnnouncementRequest,
	never
>("announcements");

export const cancelAnnouncement = async (
	id: number,
): Promise<ServiceResponse<Announcement>> =>
	httpService.post<Announcement>(
		`/backoffice/announcements/${id}/cancel`,
		{},
		{ notifySuccess: true, notifyError: true },
	);
