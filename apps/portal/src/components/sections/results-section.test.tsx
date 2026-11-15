import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
	downloadResultFile,
	getResults,
	type ResultOrder,
} from "@/api/portal-service";
import { saveBlob } from "@/lib/save-blob";
import { ResultsSection } from "./results-section";

vi.mock("@/api/portal-service", () => ({
	getResults: vi.fn(),
	downloadResultFile: vi.fn(),
}));
vi.mock("@/lib/save-blob", () => ({ saveBlob: vi.fn() }));

const ok = <T,>(data: T) => ({
	success: true as const,
	code: 200,
	message: "",
	data,
});
const fail = () => ({ success: false as const, code: 500, message: "" });

const thisYear = new Date().getFullYear();

const order = (over: Partial<ResultOrder>): ResultOrder => ({
	id: 1,
	number: "ORD-000001",
	date: `${thisYear}-02-03`,
	doctor_name: "LUIS DE LA CRUZ",
	items: [
		{
			id: 10,
			name: "Hemograma",
			reviewed_at: `${thisYear}-02-04T10:00:00Z`,
			files: [
				{
					id: 100,
					file_name: "hemograma.pdf",
					content_type: "application/pdf",
				},
				{ id: 101, file_name: "anexo.png", content_type: "image/png" },
			],
		},
	],
	...over,
});

describe("ResultsSection", () => {
	beforeEach(() => vi.clearAllMocks());

	it("lists each order with its doctor, exams and files", async () => {
		vi.mocked(getResults).mockResolvedValue(ok([order({})]));
		render(<ResultsSection />);
		expect(await screen.findByText("*portal.results.order*")).toBeVisible();
		expect(screen.getByText("Luis de la Cruz")).toBeVisible();
		expect(screen.getByText("Hemograma")).toBeVisible();
		expect(
			screen.getByRole("button", { name: /hemograma\.pdf/ }),
		).toBeVisible();
		expect(screen.getByRole("button", { name: /anexo\.png/ })).toBeVisible();
	});

	it("downloads one file by id and saves it with its name", async () => {
		const blob = new Blob(["png"]);
		vi.mocked(getResults).mockResolvedValue(ok([order({})]));
		vi.mocked(downloadResultFile).mockResolvedValue(ok(blob));
		const user = userEvent.setup();
		render(<ResultsSection />);
		await user.click(await screen.findByRole("button", { name: /anexo\.png/ }));
		expect(downloadResultFile).toHaveBeenCalledWith(101);
		expect(saveBlob).toHaveBeenCalledWith(blob, "anexo.png");
	});

	it("shows the empty and error states", async () => {
		vi.mocked(getResults).mockResolvedValueOnce(ok([]));
		const { unmount } = render(<ResultsSection />);
		expect(await screen.findByText("*portal.results.empty*")).toBeVisible();
		unmount();
		vi.mocked(getResults).mockResolvedValueOnce(fail());
		render(<ResultsSection />);
		expect(await screen.findByText("*portal.section.error*")).toBeVisible();
	});
});
