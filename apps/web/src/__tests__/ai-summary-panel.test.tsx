import { useMessageStore } from "@pengi/shared";
import { UiTextProvider } from "@pengi/ui";
import {
	act,
	fireEvent,
	render,
	screen,
	waitFor,
} from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
	getAIStatus: vi.fn(),
	getPatientAISummary: vi.fn(),
	generatePatientAISummary: vi.fn(),
	features: { ai: true } as { ai?: boolean },
	allowed: true,
}));

vi.mock("@/api/ai-service", () => ({
	getAIStatus: mocks.getAIStatus,
	getPatientAISummary: mocks.getPatientAISummary,
	generatePatientAISummary: mocks.generatePatientAISummary,
}));
vi.mock("@/hooks/use-enabled-features", () => ({
	useEnabledFeatures: () => mocks.features,
}));
vi.mock("@/hooks/use-permission", () => ({
	default: () => ({ checkPermission: () => mocks.allowed }),
}));

import { AISummaryPanel } from "@/components/features/ai/ai-summary-panel";

const status = (enabled: boolean, configured = true) => ({
	success: true,
	data: {
		configured,
		enabled,
		quotas: [
			{
				function: "summary",
				used: 3,
				limit: 300,
				period_start: "",
				period_end: "",
			},
		],
	},
});

const summary = {
	patient_id: 7,
	content: {
		overview: "Paciente con HTA controlada.",
		active_problems: [{ text: "Hipertensión arterial", sources: [11] }],
		background: "",
		allergies: "Penicilina",
		current_medications: [],
		relevant_exams: [],
		recent_evolution: "",
		alerts: [],
	},
	sources: [
		{
			label: "C1",
			medical_record_id: 11,
			date: "2026-06-12T00:00:00Z",
			motive: "Control",
		},
	],
	included_consultations: 1,
	omitted_consultations: 0,
	model: "m",
	generated_at: "2026-10-31T10:00:00Z",
	cached: false,
	ai_draft: true,
};

function renderPanel() {
	return render(
		<UiTextProvider value={{ textGet: (key) => key }}>
			<MemoryRouter>
				<AISummaryPanel patientId={7} />
			</MemoryRouter>
		</UiTextProvider>,
	);
}

describe("AISummaryPanel", () => {
	beforeEach(() => {
		act(() => {
			useMessageStore.setState({ lang: "es", messages: {} });
		});
		mocks.getAIStatus.mockReset();
		mocks.getPatientAISummary.mockReset();
		mocks.generatePatientAISummary.mockReset();
		mocks.features = { ai: true };
		mocks.allowed = true;
	});

	it("is hidden when the plan has no AI", () => {
		mocks.features = {};
		const { container } = renderPanel();
		expect(container).toBeEmptyDOMElement();
		expect(mocks.getAIStatus).not.toHaveBeenCalled();
	});

	it("is hidden until the clinic opts in", async () => {
		mocks.getAIStatus.mockResolvedValue(status(false));
		const { container } = renderPanel();
		await waitFor(() => expect(mocks.getAIStatus).toHaveBeenCalled());
		expect(container).toBeEmptyDOMElement();
		expect(mocks.getPatientAISummary).not.toHaveBeenCalled();
	});

	it("is hidden when the server has no AI key", async () => {
		mocks.getAIStatus.mockResolvedValue(status(true, false));
		mocks.getPatientAISummary.mockResolvedValue({ success: true, data: null });
		const { container } = renderPanel();
		await waitFor(() => expect(mocks.getPatientAISummary).toHaveBeenCalled());
		expect(container).toBeEmptyDOMElement();
	});

	it("still shows a cached summary when the server has no AI key", async () => {
		mocks.getAIStatus.mockResolvedValue(status(true, false));
		mocks.getPatientAISummary.mockResolvedValue({
			success: true,
			data: summary,
		});
		renderPanel();
		expect(
			await screen.findByText("Paciente con HTA controlada."),
		).toBeInTheDocument();
	});

	it("shows the API error inline when generating fails", async () => {
		mocks.getAIStatus.mockResolvedValue(status(true));
		mocks.getPatientAISummary.mockResolvedValue({ success: true, data: null });
		mocks.generatePatientAISummary.mockResolvedValue({
			success: false,
			data: {
				error_code: "E-AI-001",
				error_message: "La IA no está configurada.",
			},
		});
		renderPanel();
		fireEvent.click(
			await screen.findByRole("button", { name: /ai\.summary\.generate/ }),
		);
		expect(
			await screen.findByText("La IA no está configurada."),
		).toBeInTheDocument();
	});

	it("renders the draft with links to the source consultations", async () => {
		mocks.getAIStatus.mockResolvedValue(status(true));
		mocks.getPatientAISummary.mockResolvedValue({ success: true, data: null });
		mocks.generatePatientAISummary.mockResolvedValue({
			success: true,
			data: summary,
		});
		renderPanel();
		fireEvent.click(
			await screen.findByRole("button", { name: /ai\.summary\.generate/ }),
		);
		expect(
			await screen.findByText("Paciente con HTA controlada."),
		).toBeInTheDocument();
		expect(screen.getByText(/ai\.summary\.draft_badge/)).toBeInTheDocument();
		const link = screen.getByRole("link", { name: /ai\.summary\.source/ });
		expect(link).toHaveAttribute("href", "/clinical/medical-records/view/11");
		expect(mocks.generatePatientAISummary).toHaveBeenCalledWith(7);
	});
});
