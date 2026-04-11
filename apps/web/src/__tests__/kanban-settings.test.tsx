import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { KanbanSettings } from "@/sections/settings/kanban-settings";

// Create saveSettings mock at module level so all tests can reference it
const mockSaveSettings = vi.fn();

vi.mock("@/store/tenant-settings-store", () => ({
	useTenantSettingsStore: () => ({
		settings: {
			clinical: {
				show_next_appointment: true,
				show_diagnosis: true,
				show_medic: true,
				show_insurance: true,
				show_vital_signs: true,
				show_diagnoses: true,
				diagnosis_system: "cie11",
				patient_age_input: false,
			},
			kanban: { auto_archive_delay: "never" },
		},
		saveSettings: mockSaveSettings,
	}),
}));

vi.mock("@/hooks/use-text", () => ({
	useText: () => ({
		textGet: (key: string) => key,
	}),
}));

describe("KanbanSettings", () => {
	beforeEach(() => {
		mockSaveSettings.mockClear();
	});

	it("renders archive delay radio buttons", () => {
		render(<KanbanSettings />);

		expect(
			screen.getByText("kanban.settings.archive_delay.never"),
		).toBeDefined();
		expect(
			screen.getByText("kanban.settings.archive_delay.1_day"),
		).toBeDefined();
		expect(
			screen.getByText("kanban.settings.archive_delay.1_week"),
		).toBeDefined();
		expect(
			screen.getByText("kanban.settings.archive_delay.2_weeks"),
		).toBeDefined();
		expect(
			screen.getByText("kanban.settings.archive_delay.1_month"),
		).toBeDefined();
	});

	it("selects the correct default option", () => {
		render(<KanbanSettings />);

		const neverOption = screen.getByDisplayValue("never") as HTMLInputElement;
		expect(neverOption.checked).toBe(true);
	});

	it("calls saveSettings when option changes", async () => {
		render(<KanbanSettings />);

		const oneDayOption = screen.getByDisplayValue("1_day");
		fireEvent.click(oneDayOption);

		// Wait for async saveSettings call
		await waitFor(() => {
			expect(mockSaveSettings).toHaveBeenCalled();
		});

		// Verify the right settings were passed
		expect(mockSaveSettings).toHaveBeenCalledWith(
			expect.objectContaining({
				kanban: expect.objectContaining({
					auto_archive_delay: "1_day",
				}),
			}),
		);
	});
});
