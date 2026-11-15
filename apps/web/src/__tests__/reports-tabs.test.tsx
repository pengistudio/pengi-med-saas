import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type {
	AppointmentsReport,
	ClinicalReport,
	ReceivablesReport,
	ReportFilters,
	RevenueReport,
} from "@/api/reports-service";
import { AppointmentsTab } from "@/pages/reports/appointments-tab";
import { ClinicalTab } from "@/pages/reports/clinical-tab";
import { ReceivablesTab } from "@/pages/reports/receivables-tab";
import ReportsPage from "@/pages/reports/reports-page";
import { RevenueTab } from "@/pages/reports/revenue-tab";
import { renderWithProviders } from "@/test/test-wrapper";

const mockRevenue = vi.fn();
const mockReceivables = vi.fn();
const mockAppointments = vi.fn();
const mockClinical = vi.fn();
const mockDownload = vi.fn();
const mockDoctors = vi.fn();
const mockSaveBlob = vi.fn();

vi.mock("@/api/reports-service", async (importOriginal) => ({
	...(await importOriginal<typeof import("@/api/reports-service")>()),
	getRevenueReport: (...a: unknown[]) => mockRevenue(...a),
	getReceivablesReport: (...a: unknown[]) => mockReceivables(...a),
	getAppointmentsReport: (...a: unknown[]) => mockAppointments(...a),
	getClinicalReport: (...a: unknown[]) => mockClinical(...a),
	downloadReport: (...a: unknown[]) => mockDownload(...a),
}));

vi.mock("@/api/doctors-service", async (importOriginal) => ({
	...(await importOriginal<typeof import("@/api/doctors-service")>()),
	getDoctors: (...a: unknown[]) => mockDoctors(...a),
}));

vi.mock("@/pages/reports/report-utils", async (importOriginal) => ({
	...(await importOriginal<typeof import("@/pages/reports/report-utils")>()),
	saveBlob: (...a: unknown[]) => mockSaveBlob(...a),
}));

const filters: ReportFilters = { from: "2026-11-01", to: "2026-11-14" };
const echo = { from: filters.from, to: filters.to, doctor_id: null };
const ok = (data: unknown) => Promise.resolve({ success: true, data });
const fail = () => Promise.resolve({ success: false });

const revenue: RevenueReport = {
	filters: echo,
	totals: {
		invoiced: 1500,
		collected: 900,
		receivable: 600,
		invoice_count: 7,
		payment_count: 5,
	},
	granularity: "day",
	series: [{ period: "2026-11-02", invoiced: 100, collected: 50 }],
	by_method: [{ method: "cash", total: 900, count: 5 }],
	by_doctor: [
		{
			doctor_id: 1,
			doctor_name: "Dra. Revenue",
			invoiced: 1500,
			collected: 900,
		},
	],
};

const receivables: ReceivablesReport = {
	filters: echo,
	totals: { count: 1, total: 300, paid: 100, balance: 200 },
	buckets: [
		{ key: "0_30", count: 1, balance: 200 },
		{ key: "31_60", count: 0, balance: 0 },
		{ key: "61_plus", count: 0, balance: 0 },
	],
	items: [
		{
			invoice_id: 11,
			invoice_number: "001-001-000000123",
			patient_id: 4,
			patient_name: "Ana Receivable",
			issue_date: "2026-11-03",
			total: 300,
			paid: 100,
			balance: 200,
			age_days: 11,
			payment_status: "partial",
		},
	],
};

const appointments: AppointmentsReport = {
	filters: echo,
	totals: {
		total: 42,
		completed: 30,
		cancelled: 5,
		no_show: 7,
		no_show_rate: 0.1667,
	},
	patients: { new: 9, recurring: 33 },
	by_status: [{ status: "completed", count: 30 }],
	by_doctor: [
		{
			doctor_id: 1,
			doctor_name: "Dr. Appointments",
			total: 42,
			completed: 30,
			cancelled: 5,
			no_show: 7,
		},
	],
	by_type: [{ type_id: 1, type_name: "Consulta", count: 42 }],
	series: [{ day: "2026-11-02", total: 3, no_show: 1 }],
};

const clinical: ClinicalReport = {
	filters: echo,
	totals: { consultations: 20, prescriptions: 12, exam_orders: 4 },
	top_diagnoses: [{ code: "J00", title: "Rinofaringitis aguda", count: 6 }],
	by_doctor: [
		{
			doctor_id: 1,
			doctor_name: "Dr. Clinical",
			consultations: 20,
			prescriptions: 12,
			exam_orders: 4,
		},
	],
};

const emptyAppointments: AppointmentsReport = {
	...appointments,
	totals: { total: 0, completed: 0, cancelled: 0, no_show: 0, no_show_rate: 0 },
	patients: { new: 0, recurring: 0 },
	by_status: [],
	by_doctor: [],
	by_type: [],
	series: [],
};

beforeEach(() => {
	vi.clearAllMocks();
	// recharts measures its container.
	vi.stubGlobal(
		"ResizeObserver",
		class {
			observe() {}
			unobserve() {}
			disconnect() {}
		},
	);
	mockDoctors.mockReturnValue(ok([]));
});

describe("report tabs", () => {
	it("revenue renders the service data", async () => {
		mockRevenue.mockReturnValue(ok(revenue));
		renderWithProviders(<RevenueTab filters={filters} />);
		expect(await screen.findByText("Dra. Revenue")).toBeInTheDocument();
		expect(mockRevenue).toHaveBeenCalledWith(filters);
		expect(screen.getByText("$600,00")).toBeInTheDocument();
	});

	it("revenue shows the error state", async () => {
		mockRevenue.mockReturnValue(fail());
		renderWithProviders(<RevenueTab filters={filters} />);
		expect(await screen.findByRole("alert")).toBeInTheDocument();
	});

	it("receivables renders rows and buckets", async () => {
		mockReceivables.mockReturnValue(ok(receivables));
		renderWithProviders(<ReceivablesTab filters={filters} />);
		expect(await screen.findByText("Ana Receivable")).toBeInTheDocument();
		expect(screen.getByText("001-001-000000123")).toBeInTheDocument();
		expect(screen.getByText("11")).toBeInTheDocument();
	});

	it("receivables shows the empty state without invoices", async () => {
		mockReceivables.mockReturnValue(
			ok({
				...receivables,
				items: [],
				totals: { count: 0, total: 0, paid: 0, balance: 0 },
			}),
		);
		renderWithProviders(<ReceivablesTab filters={filters} />);
		await waitFor(() =>
			expect(screen.queryByText("Ana Receivable")).not.toBeInTheDocument(),
		);
		expect(screen.queryByRole("alert")).not.toBeInTheDocument();
		expect(screen.queryByRole("table")).not.toBeInTheDocument();
	});

	it("receivables shows the error state", async () => {
		mockReceivables.mockReturnValue(fail());
		renderWithProviders(<ReceivablesTab filters={filters} />);
		expect(await screen.findByRole("alert")).toBeInTheDocument();
	});

	it("appointments renders totals and the doctor table", async () => {
		mockAppointments.mockReturnValue(ok(appointments));
		renderWithProviders(<AppointmentsTab filters={filters} />);
		expect(await screen.findByText("Dr. Appointments")).toBeInTheDocument();
		expect(screen.getByText("16.7%")).toBeInTheDocument();
		expect(screen.getByText("33")).toBeInTheDocument();
	});

	it("appointments shows empty states for every card", async () => {
		mockAppointments.mockReturnValue(ok(emptyAppointments));
		renderWithProviders(<AppointmentsTab filters={filters} />);
		expect(await screen.findByText("0.0%")).toBeInTheDocument();
		expect(screen.queryByRole("table")).not.toBeInTheDocument();
		expect(screen.queryByRole("alert")).not.toBeInTheDocument();
	});

	it("appointments shows the error state", async () => {
		mockAppointments.mockReturnValue(fail());
		renderWithProviders(<AppointmentsTab filters={filters} />);
		expect(await screen.findByRole("alert")).toBeInTheDocument();
	});

	it("clinical renders diagnoses and doctors", async () => {
		mockClinical.mockReturnValue(ok(clinical));
		renderWithProviders(<ClinicalTab filters={filters} />);
		expect(await screen.findByText("Dr. Clinical")).toBeInTheDocument();
		expect(screen.getAllByText("J00").length).toBeGreaterThan(0);
		expect(screen.getByText("Rinofaringitis aguda")).toBeInTheDocument();
	});

	it("clinical shows the error state", async () => {
		mockClinical.mockReturnValue(fail());
		renderWithProviders(<ClinicalTab filters={filters} />);
		expect(await screen.findByRole("alert")).toBeInTheDocument();
	});
});

describe("ReportsPage exports", () => {
	beforeEach(() => {
		mockRevenue.mockReturnValue(ok(revenue));
	});

	it("CSV and PDF buttons call downloadReport and save the blob", async () => {
		const blob = new Blob(["a,b"]);
		mockDownload.mockReturnValue(ok(blob));
		renderWithProviders(<ReportsPage />);
		await screen.findByText("Dra. Revenue");

		const buttons = screen.getAllByRole("button").slice(0, 2);
		await userEvent.click(buttons[0]);
		await waitFor(() => expect(mockDownload).toHaveBeenCalledTimes(1));
		expect(mockDownload.mock.calls[0][0]).toBe("revenue");
		expect(mockDownload.mock.calls[0][1]).toBe("csv");
		await waitFor(() => expect(mockSaveBlob).toHaveBeenCalledTimes(1));
		expect(mockSaveBlob.mock.calls[0][0]).toBe(blob);
		expect(mockSaveBlob.mock.calls[0][1]).toMatch(/^reports-revenue-.*\.csv$/);

		await userEvent.click(buttons[1]);
		await waitFor(() => expect(mockDownload).toHaveBeenCalledTimes(2));
		expect(mockDownload.mock.calls[1][1]).toBe("pdf");
	});

	it("does not save a file when the download fails", async () => {
		mockDownload.mockReturnValue(fail());
		renderWithProviders(<ReportsPage />);
		await screen.findByText("Dra. Revenue");
		await userEvent.click(screen.getAllByRole("button")[0]);
		await waitFor(() => expect(mockDownload).toHaveBeenCalled());
		expect(mockSaveBlob).not.toHaveBeenCalled();
	});
});
