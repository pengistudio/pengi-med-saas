import { useMessageStore } from "@pengi/shared";
import { UiTextProvider } from "@pengi/ui";
import {
	act,
	fireEvent,
	render,
	screen,
	waitFor,
} from "@testing-library/react";
import { useForm } from "react-hook-form";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
	getAIStatus: vi.fn(),
	organizeNotesWithAI: vi.fn(),
	features: { ai: true } as { ai?: boolean },
	allowed: true,
}));

vi.mock("@/api/ai-service", () => ({
	getAIStatus: mocks.getAIStatus,
	organizeNotesWithAI: mocks.organizeNotesWithAI,
}));
vi.mock("@/hooks/use-enabled-features", () => ({
	useEnabledFeatures: () => mocks.features,
}));
vi.mock("@/hooks/use-permission", () => ({
	default: () => ({ checkPermission: () => mocks.allowed }),
}));

import {
	AIScribeCard,
	SoapDraftTag,
	type SoapValues,
	toScribeVitals,
} from "@/components/features/ai/ai-scribe-card";

const status = (enabled: boolean, scribe: boolean) => ({
	success: true,
	data: {
		configured: true,
		enabled,
		quotas: [
			{
				function: "scribe",
				used: 1,
				limit: 300,
				period_start: "",
				period_end: "",
			},
		],
		functions: { summary: true, scribe },
	},
});

const draft = {
	subjective: "Odinofagia de 2 días.",
	objective: "Exudado amigdalino.",
	assessment: "Faringitis aguda.",
	plan: "",
	suggested_diagnoses: [
		{ term: "faringitis aguda", code: "CA02.Z", title: "Faringitis aguda" },
	],
	diagnosis_system: "cie11",
	model: "m",
	ai_draft: true,
};

function renderCard(soap: SoapValues = {}, withDiagnoses = true) {
	const applySoap = vi.fn();
	const onAddDiagnosis = vi.fn();
	const view = render(
		<UiTextProvider value={{ textGet: (key) => key }}>
			<AIScribeCard
				patientId={7}
				getSoap={() => soap}
				applySoap={applySoap}
				getContext={() => ({ motive: "Dolor de garganta" })}
				diagnoses={withDiagnoses ? [] : undefined}
				onAddDiagnosis={withDiagnoses ? onAddDiagnosis : undefined}
			/>
		</UiTextProvider>,
	);
	return { ...view, applySoap, onAddDiagnosis };
}

// The card starts collapsed: a slim row that opens in place.
async function openCard() {
	const toggle = await screen.findByRole("button", {
		name: /ai\.scribe\.title/,
	});
	if (toggle.getAttribute("aria-expanded") !== "true") fireEvent.click(toggle);
}

async function organize(notes = "odinofagia, exudado, faringitis") {
	await openCard();
	fireEvent.change(await screen.findByLabelText(/ai\.scribe\.notes/), {
		target: { value: notes },
	});
	fireEvent.click(screen.getByRole("button", { name: /ai\.scribe\.organize/ }));
}

describe("AIScribeCard", () => {
	beforeEach(() => {
		act(() => {
			useMessageStore.setState({ lang: "es", messages: {} });
		});
		mocks.getAIStatus.mockReset();
		mocks.organizeNotesWithAI.mockReset();
		mocks.features = { ai: true };
		mocks.allowed = true;
	});

	it("is hidden without AI in the plan or without the permissions", () => {
		mocks.features = {};
		const { container, unmount } = renderCard();
		expect(container).toBeEmptyDOMElement();
		unmount();
		mocks.features = { ai: true };
		mocks.allowed = false;
		expect(renderCard().container).toBeEmptyDOMElement();
		expect(mocks.getAIStatus).not.toHaveBeenCalled();
	});

	it("starts collapsed and opens in place", async () => {
		mocks.getAIStatus.mockResolvedValue(status(true, true));
		renderCard();
		const toggle = await screen.findByRole("button", {
			name: /ai\.scribe\.title/,
		});
		expect(toggle).toHaveAttribute("aria-expanded", "false");
		expect(screen.queryByLabelText(/ai\.scribe\.notes/)).toBeNull();
		fireEvent.click(toggle);
		expect(toggle).toHaveAttribute("aria-expanded", "true");
		expect(screen.getByLabelText(/ai\.scribe\.notes/)).toBeInTheDocument();
	});

	it("is hidden until the clinic opts in and while the scribe has no key", async () => {
		mocks.getAIStatus.mockResolvedValue(status(false, true));
		const off = renderCard();
		await waitFor(() => expect(mocks.getAIStatus).toHaveBeenCalled());
		expect(off.container).toBeEmptyDOMElement();
		off.unmount();

		mocks.getAIStatus.mockResolvedValue(status(true, false));
		const noKey = renderCard();
		await waitFor(() => expect(mocks.getAIStatus).toHaveBeenCalledTimes(2));
		expect(noKey.container).toBeEmptyDOMElement();
	});

	it("fills empty fields directly and sends the context", async () => {
		mocks.getAIStatus.mockResolvedValue(status(true, true));
		mocks.organizeNotesWithAI.mockResolvedValue({ success: true, data: draft });
		const { applySoap } = renderCard();
		await organize();
		await waitFor(() =>
			expect(applySoap).toHaveBeenCalledWith({
				subjective: "Odinofagia de 2 días.",
				objective: "Exudado amigdalino.",
				assessment: "Faringitis aguda.",
			}),
		);
		expect(mocks.organizeNotesWithAI).toHaveBeenCalledWith({
			patient_id: 7,
			notes: "odinofagia, exudado, faringitis",
			motive: "Dolor de garganta",
			vitals: undefined,
		});
		expect(screen.getByText(/ai\.scribe\.applied/)).toBeInTheDocument();
	});

	it("asks before replacing non-empty fields and can keep them", async () => {
		mocks.getAIStatus.mockResolvedValue(status(true, true));
		mocks.organizeNotesWithAI.mockResolvedValue({ success: true, data: draft });
		const { applySoap } = renderCard({ subjective: "Lo que escribí" });
		await organize();
		expect(
			await screen.findByText(/ai\.scribe\.confirm\.description/),
		).toBeInTheDocument();
		expect(applySoap).not.toHaveBeenCalled();
		fireEvent.click(
			screen.getByRole("button", { name: /ai\.scribe\.confirm\.keep/ }),
		);
		expect(applySoap).toHaveBeenCalledWith({
			objective: "Exudado amigdalino.",
			assessment: "Faringitis aguda.",
		});
	});

	it("replaces after confirming", async () => {
		mocks.getAIStatus.mockResolvedValue(status(true, true));
		mocks.organizeNotesWithAI.mockResolvedValue({ success: true, data: draft });
		const { applySoap } = renderCard({ subjective: "Lo que escribí" });
		await organize();
		fireEvent.click(
			await screen.findByRole("button", {
				name: /ai\.scribe\.confirm\.replace/,
			}),
		);
		expect(applySoap).toHaveBeenCalledWith(
			expect.objectContaining({ subjective: "Odinofagia de 2 días." }),
		);
	});

	it("adds a suggested diagnosis only on click", async () => {
		mocks.getAIStatus.mockResolvedValue(status(true, true));
		mocks.organizeNotesWithAI.mockResolvedValue({ success: true, data: draft });
		const { onAddDiagnosis } = renderCard();
		await organize();
		const chip = await screen.findByRole("button", { name: /CA02\.Z/ });
		expect(onAddDiagnosis).not.toHaveBeenCalled();
		fireEvent.click(chip);
		expect(onAddDiagnosis).toHaveBeenCalledWith({
			code: "CA02.Z",
			title: "Faringitis aguda",
		});
	});

	it("shows the API error inline (quota, not configured, provider)", async () => {
		mocks.getAIStatus.mockResolvedValue(status(true, true));
		mocks.organizeNotesWithAI.mockResolvedValue({
			success: false,
			data: {
				error_code: "E-AI-003",
				error_message: "Límite mensual alcanzado.",
			},
		});
		const { applySoap } = renderCard();
		await organize();
		expect(
			await screen.findByText("Límite mensual alcanzado."),
		).toBeInTheDocument();
		expect(applySoap).not.toHaveBeenCalled();
	});

	it("turns typed vital signs into context", () => {
		expect(
			toScribeVitals({
				weight: "70.5",
				blood_pressure: " 120/80 ",
				heart_rate: "72",
				temperature: "",
			}),
		).toEqual({
			weight: 70.5,
			height: undefined,
			blood_pressure: "120/80",
			temperature: undefined,
			heart_rate: 72,
			o2_saturation: undefined,
		});
		expect(toScribeVitals({ weight: "" })).toBeUndefined();
	});
});

function TagHarness({
	applied,
	value,
}: {
	applied: SoapValues;
	value: string;
}) {
	const { control } = useForm({
		defaultValues: { soap_record: { subjective: value } },
	});
	return (
		<UiTextProvider value={{ textGet: (key) => key }}>
			<SoapDraftTag control={control} name="subjective" applied={applied} />
		</UiTextProvider>
	);
}

describe("SoapDraftTag", () => {
	it("shows while the text is what the model wrote", () => {
		render(
			<TagHarness
				applied={{ subjective: "Odinofagia." }}
				value="Odinofagia."
			/>,
		);
		expect(screen.getByText(/ai\.summary\.draft_badge/)).toBeInTheDocument();
	});

	it("goes away once the doctor edits the text", () => {
		render(
			<TagHarness
				applied={{ subjective: "Odinofagia." }}
				value="Odinofagia de 2 días."
			/>,
		);
		expect(screen.queryByText(/ai\.summary\.draft_badge/)).toBeNull();
	});

	it("is not shown for sections the model did not fill", () => {
		render(<TagHarness applied={{}} value="Texto propio." />);
		expect(screen.queryByText(/ai\.summary\.draft_badge/)).toBeNull();
	});
});
