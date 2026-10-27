import { createHttpService, type ServiceResponse } from "@pengi/shared";
import { apiWithTenant } from ".";

const portalSettingsService = createHttpService(apiWithTenant);

export type PortalSections = {
	appointments: boolean;
	results: boolean;
	documents: boolean;
	invoices: boolean;
};

/** How patients pick who sees them when booking from the portal. */
export type PortalBookingChooseBy = "doctor" | "specialty";

/** Hours a request waits for the clinic before it lapses. */
export const REQUEST_EXPIRY_HOURS = [12, 24, 48] as const;

/** Booking from the portal. Effective only with the portal and its appointments section on. */
/** Gap between offered start times; 0 steps by the type's duration. */
export const SLOT_INTERVAL_MINUTES = [0, 10, 15, 20, 30, 60] as const;

export type PortalBooking = {
	enabled: boolean;
	choose_by: PortalBookingChooseBy;
	/** Requests become appointments without the clinic approving them. */
	auto_confirm: boolean;
	request_expiry_hours: (typeof REQUEST_EXPIRY_HOURS)[number];
	/** 0..72: the earliest bookable time is now + this. */
	min_notice_hours: number;
	/** 1..180: the last bookable day is today + this. */
	max_days_ahead: number;
	/** 0..168: the patient may cancel up to this many hours before. */
	cancel_notice_hours: number;
	/** Minutes between offered start times; 0 = the appointment type's duration. */
	slot_interval_minutes: (typeof SLOT_INTERVAL_MINUTES)[number];
};

export const DEFAULT_PORTAL_BOOKING: PortalBooking = {
	enabled: false,
	choose_by: "doctor",
	auto_confirm: false,
	request_expiry_hours: 24,
	min_notice_hours: 2,
	max_days_ahead: 60,
	cancel_notice_hours: 24,
	slot_interval_minutes: 0,
};

export type PortalSettings = {
	enabled: boolean;
	sections: PortalSections;
	booking: PortalBooking;
};

export interface PortalSettingsResponse extends PortalSettings {
	portal_url: string;
}

export const getPortalSettings = async (): Promise<
	ServiceResponse<PortalSettingsResponse>
> =>
	portalSettingsService.get<PortalSettingsResponse>("/portal-settings", {
		notifyError: true,
	});

export const updatePortalSettings = async (
	settings: PortalSettings,
): Promise<ServiceResponse<PortalSettingsResponse>> =>
	portalSettingsService.put<PortalSettingsResponse>(
		"/portal-settings",
		settings,
		{ notifySuccess: true, notifyError: true },
	);
