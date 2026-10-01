import { useMessageStore } from "@pengi/shared";
import { UiTextProvider } from "@pengi/ui";
import { act, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
	getPatientById: vi.fn(),
	getMedicalReports: vi.fn(),
	getMedicalCertificates: vi.fn(),
}));

vi.mock("@/api/clinical-service", () => ({
	...mocks,
	downloadMedicalCertificatePdf: vi.fn(),
	downloadMedicalReportPdf: vi.fn(),
	emailMedicalCertificate: vi.fn(),
	emailMedicalReport: vi.fn(),
}));
vi.mock("@/api/signature-service", () => ({
	signMedicalCertificate: vi.fn(),
	signMedicalReport: vi.fn(),
}));
vi.mock("@/hooks/use-permission", () => ({
	default: () => ({ checkPermission: () => true }),
}));
vi.mock("@/sections/template/dashboard-template", () => ({
	DashboardLayout: ({ children }: { children: ReactNode }) => children,
}));

import MedicalDocumentsListPage from "@/pages/clincal/patient/medical-documents-list";

function renderAt(url: string) {
	return render(
		<UiTextProvider value={{ textGet: (key) => key }}>
			<MemoryRouter initialEntries={[url]}>
				<Routes>
					<Route path="/clinical" element={<p>patient list</p>} />
					<Route
						path="/clinical/medical-documents"
						element={<MedicalDocumentsListPage />}
					/>
				</Routes>
			</MemoryRouter>
		</UiTextProvider>,
	);
}

describe("MedicalDocumentsListPage", () => {
	beforeEach(() => {
		act(() => {
			useMessageStore.setState({ lang: "en", messages: {} });
		});
		for (const fn of Object.values(mocks)) fn.mockReset();
		mocks.getPatientById.mockResolvedValue({
			success: true,
			data: { ID: 7, full_name: "Ana Mora" },
		});
		mocks.getMedicalReports.mockResolvedValue({ success: true, data: [] });
		mocks.getMedicalCertificates.mockResolvedValue({ success: true, data: [] });
	});

	it("goes back to the patient list without ?patient_id instead of spinning", () => {
		renderAt("/clinical/medical-documents");

		expect(screen.getByText("patient list")).toBeInTheDocument();
		expect(mocks.getMedicalReports).not.toHaveBeenCalled();
	});

	it("lists the patient's documents and stops loading", async () => {
		renderAt("/clinical/medical-documents?patient_id=7");

		expect(
			await screen.findByText("clinical.medical_documents.empty"),
		).toBeInTheDocument();
		expect(mocks.getMedicalReports).toHaveBeenCalledWith(7);
	});
});
