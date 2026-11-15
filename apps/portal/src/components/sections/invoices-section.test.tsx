import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
	downloadInvoiceRide,
	downloadInvoiceXml,
	getInvoices,
	type PortalInvoice,
} from "@/api/portal-service";
import { saveBlob } from "@/lib/save-blob";
import { InvoicesSection } from "./invoices-section";

vi.mock("@/api/portal-service", () => ({
	getInvoices: vi.fn(),
	downloadInvoiceRide: vi.fn(),
	downloadInvoiceXml: vi.fn(),
}));
vi.mock("@/lib/save-blob", () => ({ saveBlob: vi.fn() }));

const ok = <T,>(data: T) => ({
	success: true as const,
	code: 200,
	message: "",
	data,
});

const thisYear = new Date().getFullYear();

const invoice = (over: Partial<PortalInvoice>): PortalInvoice => ({
	id: 4,
	number: "001-001-000000123",
	date: `${thisYear}-05-06`,
	total: 25.5,
	access_key: "1".repeat(49),
	...over,
});

describe("InvoicesSection", () => {
	beforeEach(() => vi.clearAllMocks());

	it("lists each invoice with its number and total", async () => {
		vi.mocked(getInvoices).mockResolvedValue(ok([invoice({})]));
		render(<InvoicesSection />);
		expect(await screen.findByText("001-001-000000123")).toBeVisible();
		expect(screen.getByText(/25[.,]50/)).toBeVisible();
		expect(screen.getByText("*portal.invoices.hint*")).toBeVisible();
	});

	it("downloads the RIDE and the XML of that invoice", async () => {
		const pdf = new Blob(["%PDF"]);
		const xml = new Blob(["<factura/>"]);
		vi.mocked(getInvoices).mockResolvedValue(ok([invoice({})]));
		vi.mocked(downloadInvoiceRide).mockResolvedValue(ok(pdf));
		vi.mocked(downloadInvoiceXml).mockResolvedValue(ok(xml));
		const user = userEvent.setup();
		render(<InvoicesSection />);

		await user.click(
			await screen.findByRole("button", { name: /\*portal\.invoices\.ride\*/ }),
		);
		expect(downloadInvoiceRide).toHaveBeenCalledWith(4);
		expect(saveBlob).toHaveBeenCalledWith(pdf, "001-001-000000123.pdf");

		await user.click(
			screen.getByRole("button", { name: /\*portal\.invoices\.xml\*/ }),
		);
		expect(downloadInvoiceXml).toHaveBeenCalledWith(4);
		expect(saveBlob).toHaveBeenCalledWith(xml, "001-001-000000123.xml");
	});

	it("shows the empty state", async () => {
		vi.mocked(getInvoices).mockResolvedValue(ok([]));
		render(<InvoicesSection />);
		expect(await screen.findByText("*portal.invoices.empty*")).toBeVisible();
	});
});
