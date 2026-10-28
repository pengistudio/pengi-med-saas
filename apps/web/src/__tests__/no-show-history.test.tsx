import { screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { hasStarted } from "@/components/features/appointments/appointment-utils";
import {
	NoShowBadge,
	NoShowWarning,
} from "@/components/features/appointments/no-show-history";
import { renderWithProviders } from "@/test/test-wrapper";

const mockGetNoShows = vi.fn();

vi.mock("@/api/clinical-service", async (importOriginal) => ({
	...(await importOriginal<typeof import("@/api/clinical-service")>()),
	getPatientNoShows: (id: number) => mockGetNoShows(id),
}));

const answer = (count: number, last_date: string | null = null) => ({
	success: true,
	data: { count, last_date, months: 12 },
});

describe("NoShowWarning", () => {
	beforeEach(() => mockGetNoShows.mockReset());

	it("warns, without blocking, from two no-shows in 12 months", async () => {
		mockGetNoShows.mockResolvedValue(answer(3, "2026-10-01T05:00:00Z"));
		renderWithProviders(<NoShowWarning patientId={7} />);
		expect(await screen.findByRole("status")).toBeDefined();
		expect(mockGetNoShows).toHaveBeenCalledWith(7);
	});

	it("says nothing below the threshold", async () => {
		mockGetNoShows.mockResolvedValue(answer(1, "2026-10-01T05:00:00Z"));
		renderWithProviders(<NoShowWarning patientId={7} />);
		await waitFor(() => expect(mockGetNoShows).toHaveBeenCalled());
		expect(screen.queryByRole("status")).toBeNull();
	});

	it("asks nothing without a patient", () => {
		renderWithProviders(<NoShowWarning patientId={null} />);
		expect(mockGetNoShows).not.toHaveBeenCalled();
		expect(screen.queryByRole("status")).toBeNull();
	});
});

describe("NoShowBadge", () => {
	beforeEach(() => mockGetNoShows.mockReset());

	it("shows the count and the latest no-show", async () => {
		mockGetNoShows.mockResolvedValue(answer(1, "2026-10-01T05:00:00Z"));
		const { container } = renderWithProviders(<NoShowBadge patientId={7} />);
		await waitFor(() =>
			expect(container.textContent).toContain("appointments.no_shows.last"),
		);
	});

	it("is hidden for a patient who never missed", async () => {
		mockGetNoShows.mockResolvedValue(answer(0));
		const { container } = renderWithProviders(<NoShowBadge patientId={7} />);
		await waitFor(() => expect(mockGetNoShows).toHaveBeenCalled());
		expect(container.textContent).toBe("");
	});
});

describe("hasStarted", () => {
	const appt = { date: "2026-10-28T05:00:00Z", start_time: "09:30" };

	it("is true from the start time on", () => {
		expect(hasStarted(appt, new Date(2026, 9, 28, 9, 30))).toBe(true);
		expect(hasStarted(appt, new Date(2026, 9, 29, 8, 0))).toBe(true);
	});

	it("is false before it", () => {
		expect(hasStarted(appt, new Date(2026, 9, 28, 9, 29))).toBe(false);
		expect(hasStarted(appt, new Date(2026, 9, 27, 18, 0))).toBe(false);
	});
});
