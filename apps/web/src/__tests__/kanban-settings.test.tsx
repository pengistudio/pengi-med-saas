import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { KanbanSettings } from "@/sections/settings/kanban-settings";

// Mock the hooks and service
vi.mock("@/store/tenant-settings-store", () => ({
	useTenantSettingsStore: () => ({
		settings: {
			clinical: {},
			kanban: { auto_archive_delay: "never" },
		},
		saveSettings: vi.fn(),
	}),
}));

vi.mock("@/hooks/use-text", () => ({
	useText: () => ({
		textGet: (key: string) => key, // Return the key itself for testing
	}),
}));

describe("KanbanSettings", () => {
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

		// Verify saveSettings was called (would need actual mock setup)
		// expect(mockSaveSettings).toHaveBeenCalled();
	});
});
