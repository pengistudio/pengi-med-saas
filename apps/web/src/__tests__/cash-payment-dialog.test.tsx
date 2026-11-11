import { fireEvent, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
	type CashSessionDetail,
	createPayment,
	getCurrentSession,
	getInvoicePayments,
	type InvoicePayments,
} from "@/api/cash-service";
import { PaymentDialog } from "@/components/features/cash/payment-dialog";
import { useCashStore } from "@/store/cash-store";
import { renderWithProviders } from "@/test/test-wrapper";

vi.mock("@/api/cash-service", async (importOriginal) => ({
	...(await importOriginal<typeof import("@/api/cash-service")>()),
	getCurrentSession: vi.fn(),
	getInvoicePayments: vi.fn(),
	createPayment: vi.fn(),
}));

const ok = <T,>(data: T) => ({
	success: true as const,
	code: 200,
	message: "",
	data,
});

const openSession = { ID: 1, status: "open" } as CashSessionDetail;
const invoicePayments: InvoicePayments = {
	invoice_id: 5,
	total: 100,
	paid_amount: 40,
	balance: 60,
	payment_status: "partial",
	payments: [],
};

function renderDialog() {
	renderWithProviders(
		<MemoryRouter>
			<PaymentDialog
				invoice={{ id: 5, number: "001-001-000000005" }}
				onOpenChange={vi.fn()}
			/>
		</MemoryRouter>,
	);
}

const submit = () =>
	fireEvent.submit(
		screen.getByRole("spinbutton").closest("form") as HTMLFormElement,
	);

describe("PaymentDialog", () => {
	beforeEach(() => {
		vi.mocked(createPayment).mockReset();
		vi.mocked(getInvoicePayments).mockResolvedValue(ok(invoicePayments));
		useCashStore.setState({ session: null, loaded: false, loading: false });
	});

	it("does not let the payment exceed the balance", async () => {
		vi.mocked(getCurrentSession).mockResolvedValue(ok(openSession));
		renderDialog();

		const amount = (await screen.findByRole("spinbutton")) as HTMLInputElement;
		// Starts with the balance due.
		expect(amount.value).toBe("60");

		fireEvent.change(amount, { target: { value: "60.01" } });
		expect(
			await screen.findByText("*cash.payment.error.exceeds_balance*"),
		).toBeDefined();
		const charge = screen.getByRole("button", {
			name: "*cash.payment.submit*",
		});
		expect((charge as HTMLButtonElement).disabled).toBe(true);

		submit();
		await new Promise((r) => setTimeout(r, 50));
		expect(createPayment).not.toHaveBeenCalled();
	});

	it("charges a partial amount within the balance", async () => {
		vi.mocked(getCurrentSession).mockResolvedValue(ok(openSession));
		vi.mocked(createPayment).mockResolvedValue(
			ok({ ...invoicePayments, paid_amount: 65, balance: 35 }),
		);
		renderDialog();

		const amount = await screen.findByRole("spinbutton");
		fireEvent.change(amount, { target: { value: "25" } });
		submit();
		await waitFor(() =>
			expect(createPayment).toHaveBeenCalledWith(5, [
				{ method: "cash", amount: 25, reference: undefined },
			]),
		);
	});

	it("says the desk is closed instead of showing the form", async () => {
		vi.mocked(getCurrentSession).mockResolvedValue(ok(null));
		renderDialog();

		expect(await screen.findByText("*cash.payment.desk_closed*")).toBeDefined();
		expect(screen.queryByRole("spinbutton")).toBeNull();
		expect(screen.getByText("*cash.payment.go_to_cash*")).toBeDefined();
	});
});
