import { fireEvent, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type {
	WhatsAppBotSettings as BotSettings,
	WhatsAppBotPreview,
} from "@/api/whatsapp-service";
import {
	botReady,
	cleanFAQ,
	faqDraftValid,
	readinessWithDraft,
} from "@/lib/whatsapp-bot";
import { WhatsAppBotSettings } from "@/sections/settings/whatsapp-bot-settings";
import { renderWithProviders } from "@/test/test-wrapper";

const mocks = vi.hoisted(() => ({
	features: { ai: true } as Record<string, boolean>,
	settings: undefined as unknown as BotSettings,
	get: vi.fn(),
	update: vi.fn(),
	test: vi.fn(),
}));

vi.mock("@/hooks/use-enabled-features", () => ({
	useEnabledFeatures: () => mocks.features,
}));
vi.mock("@/api/whatsapp-service", async (importOriginal) => {
	const actual =
		await importOriginal<typeof import("@/api/whatsapp-service")>();
	return {
		...actual,
		getWhatsAppBotSettings: mocks.get,
		updateWhatsAppBotSettings: mocks.update,
		testWhatsAppBot: mocks.test,
	};
});

function readySettings(overrides: Partial<BotSettings> = {}): BotSettings {
	return {
		enabled: false,
		during_office_hours: false,
		faq: [{ question: "¿Tienen parqueadero?", answer: "Sí, gratuito." }],
		readiness: {
			plan_whatsapp: true,
			plan_ai: true,
			ai_opt_in: true,
			provider_configured: true,
			whatsapp_connected: true,
			office_hours: true,
			address: true,
			faq: true,
		},
		ready: true,
		missing_key: "",
		office_hours: "Lunes: 08:00 a 13:00",
		address: "Av. Amazonas N34-120",
		clinic_phone: "+593 99 123 4567",
		booking_link: false,
		in_office_hours: false,
		quota: { used: 3, limit: 500 },
		limits: { max_entries: 30, max_question: 200, max_answer: 500 },
		...overrides,
	};
}

const renderBot = () =>
	renderWithProviders(
		<MemoryRouter>
			<WhatsAppBotSettings />
		</MemoryRouter>,
	);

// Messages are not loaded in tests: useText renders each key as *key*.
const enabledSwitch = () =>
	screen.getByRole("switch", { name: "*settings.whatsapp.bot.enabled*" });
/** Base UI's switch is a span: disabled shows as aria-/data-disabled. */
const switchDisabled = () => {
	const el = enabledSwitch();
	return (
		el.getAttribute("aria-disabled") === "true" ||
		el.hasAttribute("data-disabled")
	);
};

describe("WhatsAppBotSettings", () => {
	beforeEach(() => {
		mocks.features = { ai: true };
		mocks.settings = readySettings();
		mocks.get.mockReset();
		mocks.get.mockImplementation(async () => ({
			success: true,
			data: mocks.settings,
		}));
		mocks.update.mockReset();
		mocks.update.mockImplementation(async (payload) => ({
			success: true,
			data: { ...mocks.settings, ...payload },
		}));
		mocks.test.mockReset();
	});

	it("renders nothing and loads nothing when the plan has no AI", () => {
		mocks.features = { ai: false };
		const { container } = renderBot();
		expect(container.textContent).toBe("");
		expect(mocks.get).not.toHaveBeenCalled();
	});

	it("explains why the switch is disabled when something is missing", async () => {
		mocks.settings = readySettings({
			ready: false,
			readiness: {
				...readySettings().readiness,
				office_hours: false,
				address: false,
			},
		});
		renderBot();
		await waitFor(() => expect(enabledSwitch()).toBeDefined());
		expect(switchDisabled()).toBe(true);
		expect(screen.getByTestId("bot-readiness-office_hours").dataset.ok).toBe(
			"false",
		);
		expect(screen.getByTestId("bot-readiness-address").dataset.ok).toBe(
			"false",
		);
		expect(screen.getByTestId("bot-readiness-faq").dataset.ok).toBe("true");
		expect(
			screen.getByText("*settings.whatsapp.bot.readiness.office_hours*"),
		).toBeDefined();
	});

	it("enables and saves with the FAQ when everything is ready", async () => {
		renderBot();
		await waitFor(() => expect(enabledSwitch()).toBeDefined());
		expect(screen.queryByTestId("bot-readiness-faq")).toBeNull();
		fireEvent.click(enabledSwitch());
		fireEvent.click(screen.getByText("*form.save*"));
		await waitFor(() =>
			expect(mocks.update).toHaveBeenCalledWith({
				enabled: true,
				during_office_hours: false,
				faq: [{ question: "¿Tienen parqueadero?", answer: "Sí, gratuito." }],
			}),
		);
	});

	it("edits the FAQ: add, count characters, remove", async () => {
		mocks.settings = readySettings({ faq: [] });
		renderBot();
		await waitFor(() =>
			expect(
				screen.getByText("*settings.whatsapp.bot.faq.empty*"),
			).toBeDefined(),
		);
		// Without a FAQ entry the switch can't be turned on.
		expect(switchDisabled()).toBe(true);

		fireEvent.click(screen.getByText("*settings.whatsapp.bot.faq.add*"));
		const question = screen.getByLabelText(
			"*settings.whatsapp.bot.faq.question*",
		);
		const answer = screen.getByLabelText("*settings.whatsapp.bot.faq.answer*");
		fireEvent.change(question, { target: { value: "¿Aceptan tarjeta?" } });
		// Half a pair can't be saved.
		expect(
			screen.getByText("*settings.whatsapp.bot.faq.invalid*"),
		).toBeDefined();
		fireEvent.change(answer, { target: { value: "Sí, débito y crédito." } });
		expect(screen.getByText("21/500")).toBeDefined();
		expect(screen.getByText("17/200")).toBeDefined();
		expect(
			screen.queryByText("*settings.whatsapp.bot.faq.invalid*"),
		).toBeNull();
		expect(switchDisabled()).toBe(false);

		fireEvent.click(
			screen.getAllByText("*settings.whatsapp.bot.faq.remove*")[0],
		);
		expect(
			screen.queryByLabelText("*settings.whatsapp.bot.faq.question*"),
		).toBeNull();
	});

	it("shows what the test panel would send", async () => {
		const preview: WhatsAppBotPreview = {
			action: "handoff",
			reason: "emergency",
			guard: "",
			message: "Si es una emergencia, llama ahora al 911.",
			model_called: false,
			in_office_hours: true,
			would_reply_now: false,
		};
		mocks.test.mockResolvedValue({ success: true, data: preview });
		renderBot();
		await waitFor(() => expect(enabledSwitch()).toBeDefined());
		const run = screen
			.getByText("*settings.whatsapp.bot.test.run*")
			.closest("button") as HTMLButtonElement;
		expect(run.disabled).toBe(true);
		fireEvent.change(
			screen.getByLabelText("*settings.whatsapp.bot.test.title*"),
			{ target: { value: "me duele el pecho" } },
		);
		fireEvent.click(run);
		await waitFor(() =>
			expect(screen.getByTestId("bot-test-result")).toBeDefined(),
		);
		expect(mocks.test).toHaveBeenCalledWith("me duele el pecho");
		expect(screen.getByText(preview.message)).toBeDefined();
		expect(
			screen.getByText("*settings.whatsapp.bot.test.handoff*"),
		).toBeDefined();
		expect(screen.getByText("*whatsapp.bot.reason.emergency*")).toBeDefined();
		expect(
			screen.getByText("*settings.whatsapp.bot.test.in_hours*"),
		).toBeDefined();
	});
});

describe("whatsapp-bot helpers", () => {
	const limits = { max_entries: 2, max_question: 10, max_answer: 10 };
	it("cleans and validates the FAQ draft", () => {
		expect(
			cleanFAQ([
				{ question: " a ", answer: " b " },
				{ question: "", answer: " " },
			]),
		).toEqual([{ question: "a", answer: "b" }]);
		expect(faqDraftValid([{ question: "a", answer: "" }], limits)).toBe(false);
		expect(faqDraftValid([{ question: "a", answer: "b" }], limits)).toBe(true);
		expect(
			faqDraftValid([{ question: "a", answer: "x".repeat(11) }], limits),
		).toBe(false);
		expect(
			faqDraftValid(
				[
					{ question: "a", answer: "b" },
					{ question: "c", answer: "d" },
					{ question: "e", answer: "f" },
				],
				limits,
			),
		).toBe(false);
	});
	it("judges readiness on the draft", () => {
		const r = readySettings().readiness;
		expect(botReady(readinessWithDraft(r, []))).toBe(false);
		expect(
			botReady(readinessWithDraft(r, [{ question: "a", answer: "b" }])),
		).toBe(true);
	});
});
