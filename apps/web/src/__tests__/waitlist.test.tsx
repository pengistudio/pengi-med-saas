import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
	slotFromSearch,
	type WaitlistCandidates,
	type WaitlistEntry,
} from "@/api/waitlist-service";
import { FreedSlotPanel } from "@/components/features/waitlist/freed-slot-panel";
import { entryPreference } from "@/components/features/waitlist/waitlist-utils";
import { renderWithProviders } from "@/test/test-wrapper";

const mockCandidates = vi.fn();
const mockOffer = vi.fn();

vi.mock("@/api/waitlist-service", async (importOriginal) => ({
	...(await importOriginal<typeof import("@/api/waitlist-service")>()),
	getWaitlistCandidates: (...args: unknown[]) => mockCandidates(...args),
	offerWaitlistSlot: (...args: unknown[]) => mockOffer(...args),
}));

const entry = (
	id: number,
	over: Partial<WaitlistEntry> = {},
): WaitlistEntry => ({
	id,
	status: "waiting",
	source: "staff",
	patient: {
		id: id * 10,
		first_name: `Paciente${id}`,
		last_name: "Pérez",
		document: "0102030405",
		phone: "0991234567",
		whatsapp_opt_in: true,
	},
	doctor: null,
	appointment_type: null,
	date_from: "",
	date_to: "",
	appointment: null,
	notes: "",
	created_at: "2026-10-20T10:00:00-05:00",
	expires_at: "2026-11-20T10:00:00-05:00",
	offer: null,
	can_whatsapp: true,
	...over,
});

const answer = (data: WaitlistCandidates) => ({ success: true, data });
const slot = {
	date: "2026-11-04",
	start_time: "10:00",
	end_time: "10:30",
	doctor_id: 3,
};

describe("slotFromSearch", () => {
	it("reads the freed slot of the notification link", () => {
		expect(
			slotFromSearch(
				new URLSearchParams(
					"date=2026-11-04&start_time=10:00&end_time=10:30&doctor_id=3&patient_id=8",
				),
			),
		).toEqual({
			date: "2026-11-04",
			start_time: "10:00",
			end_time: "10:30",
			doctor_id: 3,
			appointment_type_id: null,
			patient_id: 8,
		});
		expect(slotFromSearch(new URLSearchParams("doctor_id=3"))).toBeNull();
	});
});

describe("entryPreference", () => {
	const textGet = (key: string, params?: Record<string, string | number>) =>
		params ? `${key}(${Object.values(params).join(",")})` : key;
	const formatDate = (value: unknown) => String(value);

	it("joins the type, the range and the appointment to move earlier", () => {
		const text = entryPreference(
			entry(1, {
				appointment_type: { id: 1, name: "Consulta", duration_minutes: 30 },
				date_from: "2026-11-03",
				date_to: "2026-11-10",
				appointment: {
					id: 9,
					date: "2026-11-20",
					start_time: "09:00",
					status: "scheduled",
				},
			}),
			textGet as never,
			formatDate as never,
		);
		expect(text).toBe(
			"Consulta · waitlist.range.between(2026-11-03,2026-11-10) · waitlist.move_earlier(2026-11-20,09:00)",
		);
		expect(
			entryPreference(entry(2), textGet as never, formatDate as never),
		).toBe("waitlist.type.any");
	});
});

describe("FreedSlotPanel", () => {
	beforeEach(() => {
		mockCandidates.mockReset();
		mockOffer.mockReset();
	});

	it("lists candidates in order and offers by WhatsApp", async () => {
		mockCandidates.mockResolvedValue(
			answer({
				whatsapp_available: true,
				doctor: "Dra. Ana",
				candidates: [entry(1), entry(2, { can_whatsapp: false })],
			}),
		);
		mockOffer.mockResolvedValue({ success: true, data: entry(1) });
		const user = userEvent.setup();
		renderWithProviders(
			<FreedSlotPanel slot={slot} canManage onClose={() => {}} />,
		);

		const offers = await screen.findAllByRole("button", {
			name: /waitlist\.offer\.whatsapp/,
		});
		expect(offers).toHaveLength(2);
		expect(screen.getByText("Paciente1 Pérez")).toBeDefined();
		// Without consent or phone the WhatsApp button is disabled.
		expect((offers[1] as HTMLButtonElement).disabled).toBe(true);

		await user.click(offers[0]);
		expect(mockOffer).toHaveBeenCalledWith(1, slot, "whatsapp");
		await waitFor(() => expect(mockCandidates).toHaveBeenCalledTimes(2));
	});

	it("without WhatsApp offers only a call: shows the phone, then marks it offered", async () => {
		mockCandidates.mockResolvedValue(
			answer({ whatsapp_available: false, doctor: "", candidates: [entry(1)] }),
		);
		mockOffer.mockResolvedValue({ success: true, data: entry(1) });
		const user = userEvent.setup();
		renderWithProviders(
			<FreedSlotPanel slot={slot} canManage onClose={() => {}} />,
		);

		await user.click(
			await screen.findByRole("button", { name: /waitlist\.offer\.call/ }),
		);
		expect(
			screen.queryByRole("button", { name: /waitlist\.offer\.whatsapp/ }),
		).toBeNull();
		expect(screen.getByRole("link", { name: "0991234567" })).toBeDefined();
		await user.click(
			screen.getByRole("button", { name: /waitlist\.offer\.mark_called/ }),
		);
		expect(mockOffer).toHaveBeenCalledWith(1, slot, "phone");
	});

	it("an entry already offered this slot shows a badge, no buttons", async () => {
		mockCandidates.mockResolvedValue(
			answer({
				whatsapp_available: true,
				doctor: "",
				candidates: [entry(1, { status: "offered", offered_this_slot: true })],
			}),
		);
		renderWithProviders(
			<FreedSlotPanel slot={slot} canManage onClose={() => {}} />,
		);
		expect(await screen.findByText(/waitlist\.offer\.offered/)).toBeDefined();
		expect(
			screen.queryByRole("button", { name: /waitlist\.offer\.whatsapp/ }),
		).toBeNull();
		expect(
			screen.queryByRole("button", { name: /waitlist\.offer\.call/ }),
		).toBeNull();
	});

	it("says when nobody matches, and read-only users get no buttons", async () => {
		mockCandidates.mockResolvedValue(
			answer({ whatsapp_available: true, doctor: "", candidates: [] }),
		);
		renderWithProviders(
			<FreedSlotPanel slot={slot} canManage={false} onClose={() => {}} />,
		);
		expect(await screen.findByText(/waitlist\.slot\.empty/)).toBeDefined();
	});
});
