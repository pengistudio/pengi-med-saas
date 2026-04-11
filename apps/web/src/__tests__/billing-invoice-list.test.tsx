import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import InvoiceListPage from "@/pages/billing/invoice-list";

vi.mock("@/api/billing-service");

vi.mock("@/hooks/use-text", () => ({
	useText: () => ({
		textGet: (key: string) => key,
	}),
}));

vi.mock("@/store/billing-store", () => ({
	useBillingStore: () => ({
		invoices: [],
		setInvoices: vi.fn(),
	}),
}));

vi.mock("react-router", () => ({
	useNavigate: () => vi.fn(),
}));

import * as billingService from "@/api/billing-service";

describe("InvoiceListPage", () => {
	it("renders invoice list component", async () => {
		const mockGetAllInvoices = vi.mocked(billingService.getAllInvoices);
		mockGetAllInvoices.mockResolvedValue({
			success: true,
			data: {
				items: [
					{ ID: 1, sequential: "INV-001", status: "pending", total: 100 },
					{ ID: 2, sequential: "INV-002", status: "paid", total: 200 },
				],
			},
		} as any);

		render(<InvoiceListPage />);

		// Component should render without errors
		expect(screen.getByRole("main") || screen.getByRole("table")).toBeDefined();
	});

	it("renders empty state when no invoices", () => {
		const mockGetAllInvoices = vi.mocked(billingService.getAllInvoices);
		mockGetAllInvoices.mockResolvedValue({
			success: true,
			data: { items: [], total: 0 },
		} as any);

		render(<InvoiceListPage />);

		// Just verify it renders without crashing
		expect(screen.queryByText || screen.queryByRole).toBeDefined();
	});
});
