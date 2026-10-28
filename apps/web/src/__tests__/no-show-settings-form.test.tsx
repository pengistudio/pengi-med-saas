import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { DEFAULT_AGENDA_SETTINGS } from "@/api/settings-service";
import { NoShowSettingsForm } from "@/sections/forms/settings/no-show-settings-form";
import { renderWithProviders } from "@/test/test-wrapper";

// Messages are not loaded in tests: useText renders each key as *key*.
const toggle = () =>
	screen.getByRole("switch", { name: "*agenda.no_show.auto*" });

describe("NoShowSettingsForm", () => {
	it("is off by default, with the minutes disabled until it is on", () => {
		renderWithProviders(
			<NoShowSettingsForm
				initialData={DEFAULT_AGENDA_SETTINGS}
				onSubmit={vi.fn()}
			/>,
		);
		expect(toggle().getAttribute("aria-checked")).toBe("false");
		const minutes = screen.getByRole("spinbutton") as HTMLInputElement;
		expect(minutes.value).toBe("30");
		expect(minutes.disabled).toBe(true);
		expect(screen.getByText("*agenda.no_show.warning*")).toBeDefined();

		fireEvent.click(toggle());
		expect(minutes.disabled).toBe(false);
	});

	it("submits the flag and the minutes, never a since", async () => {
		const onSubmit = vi.fn();
		renderWithProviders(
			<NoShowSettingsForm
				initialData={DEFAULT_AGENDA_SETTINGS}
				onSubmit={onSubmit}
			/>,
		);
		fireEvent.click(toggle());
		fireEvent.change(screen.getByRole("spinbutton"), {
			target: { value: "45" },
		});
		fireEvent.click(screen.getByText("*form.save*"));
		await waitFor(() =>
			expect(onSubmit).toHaveBeenCalledWith(
				{ auto_no_show: true, no_show_after_minutes: 45 },
				expect.anything(),
			),
		);
	});

	it("rejects minutes outside 5–240", async () => {
		const onSubmit = vi.fn();
		renderWithProviders(
			<NoShowSettingsForm
				initialData={{ ...DEFAULT_AGENDA_SETTINGS, auto_no_show: true }}
				onSubmit={onSubmit}
			/>,
		);
		const minutes = screen.getByRole("spinbutton");
		fireEvent.change(minutes, { target: { value: "300" } });
		// Submit the form itself: the browser's own max check would stop a click first.
		fireEvent.submit(minutes.closest("form") as HTMLFormElement);
		await waitFor(() =>
			expect(minutes.getAttribute("aria-invalid")).toBe("true"),
		);
		expect(onSubmit).not.toHaveBeenCalled();
	});
});
