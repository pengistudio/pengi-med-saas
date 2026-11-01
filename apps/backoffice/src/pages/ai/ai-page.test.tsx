import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { AIFunctionSettings } from "@/api/ai-service";

const mocks = vi.hoisted(() => ({
	getAISettings: vi.fn(),
	getAIUsage: vi.fn(),
	updateAISettings: vi.fn(),
}));

vi.mock("@/api/ai-service", () => mocks);

import AIPage from "./ai-page";

const summary: AIFunctionSettings = {
	function: "summary",
	base_url: "",
	default_base_url: "https://inference.do-ai.run/v1",
	model: "",
	default_model: "openai-gpt-oss-120b",
	reasoning_effort: "",
	default_reasoning_effort: "",
	has_api_key: true,
	env_has_api_key: false,
	api_key_source: "db",
	encryption_available: true,
	price_input: 0,
	price_output: 0,
};

const ok = <T,>(data: T) => ({ success: true, code: 200, message: "", data });

describe("AIPage provider settings", () => {
	beforeEach(() => {
		for (const fn of Object.values(mocks)) fn.mockReset();
		mocks.getAISettings.mockResolvedValue(ok([summary]));
		mocks.getAIUsage.mockResolvedValue(ok({ month: "2026-10", rows: [] }));
		mocks.updateAISettings.mockResolvedValue(ok([summary]));
	});

	it("shows the env defaults, never a key, and sends a new key write-only", async () => {
		const user = userEvent.setup();
		render(<AIPage />);

		const key = (await screen.findByLabelText(
			/backoffice\.ai\.api_key/,
		)) as HTMLInputElement;
		expect(key.type).toBe("password");
		expect(key.value).toBe("");
		expect(screen.getByLabelText(/backoffice\.ai\.base_url/)).toHaveAttribute(
			"placeholder",
			"https://inference.do-ai.run/v1",
		);
		expect(
			screen.getByText(/backoffice\.ai\.api_key\.source\.db/),
		).toBeInTheDocument();

		await user.type(
			screen.getByLabelText(/backoffice\.ai\.base_url/),
			"http://localhost:11434/v1",
		);
		await user.type(key, "sk-new");
		await user.click(
			screen.getByRole("button", { name: /backoffice\.ai\.settings\.save/ }),
		);

		await waitFor(() => expect(mocks.updateAISettings).toHaveBeenCalled());
		expect(mocks.updateAISettings.mock.calls[0][0]).toEqual([
			expect.objectContaining({
				function: "summary",
				base_url: "http://localhost:11434/v1",
				api_key: "sk-new",
				clear_api_key: false,
			}),
		]);
		// After saving, the typed key is gone from the form.
		await waitFor(() => expect(key.value).toBe(""));
	});

	it("clears the stored key explicitly", async () => {
		const user = userEvent.setup();
		render(<AIPage />);
		await user.click(
			await screen.findByRole("button", {
				name: /backoffice\.ai\.api_key\.clear/,
			}),
		);
		await user.click(
			screen.getByRole("button", { name: /backoffice\.ai\.settings\.save/ }),
		);
		await waitFor(() => expect(mocks.updateAISettings).toHaveBeenCalled());
		expect(mocks.updateAISettings.mock.calls[0][0][0]).toMatchObject({
			api_key: "",
			clear_api_key: true,
		});
	});
});
