import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { closeSession } from "@/api/cash-service";
import { CloseSessionDialog } from "@/components/features/cash/close-session-dialog";
import { renderWithProviders } from "@/test/test-wrapper";

vi.mock("@/api/cash-service", async (importOriginal) => ({
	...(await importOriginal<typeof import("@/api/cash-service")>()),
	closeSession: vi.fn(),
}));

const closeMock = vi.mocked(closeSession);

const noteRequired = {
	success: false as const,
	code: 422,
	message: "*cash.session.error.note_required*",
	data: { error_code: "E-CASH-005", error_message: "" },
};

function renderDialog(onClosed = vi.fn()) {
	renderWithProviders(
		<CloseSessionDialog
			sessionId={7}
			open
			onOpenChange={vi.fn()}
			onClosed={onClosed}
		/>,
	);
	return onClosed;
}

const counted = () => screen.getByRole("spinbutton") as HTMLInputElement;
const submit = () =>
	fireEvent.submit(counted().closest("form") as HTMLFormElement);

describe("CloseSessionDialog (blind close)", () => {
	beforeEach(() => closeMock.mockReset());

	it("asks only for the counted cash: no expected amount, totals or float", () => {
		renderDialog();
		// One number field (the count) and no amount anywhere in the dialog.
		expect(screen.getAllByRole("spinbutton")).toHaveLength(1);
		const dialog = screen.getByRole("dialog");
		expect(dialog.textContent).not.toMatch(/\$|\d+[.,]\d{2}/);
		expect(screen.queryByText("*cash.summary.expected*")).toBeNull();
		expect(screen.queryByText("*cash.summary.opening_float*")).toBeNull();
		expect(screen.queryByText("*cash.close.note_required*")).toBeNull();
	});

	it("asks for the note on E-CASH-005 and resends with it", async () => {
		closeMock.mockResolvedValueOnce(noteRequired);
		const onClosed = renderDialog();

		fireEvent.change(counted(), { target: { value: "48.5" } });
		submit();
		await waitFor(() =>
			expect(closeMock).toHaveBeenCalledWith(7, {
				counted_cash: 48.5,
				note: undefined,
			}),
		);
		expect(await screen.findByText("*cash.close.note_required*")).toBeDefined();
		// Still nothing about the expected amount after the mismatch.
		expect(screen.queryByText("*cash.summary.expected*")).toBeNull();
		expect(onClosed).not.toHaveBeenCalled();

		// The note is now required: submitting without it doesn't call the API.
		submit();
		await waitFor(() =>
			expect(screen.getByRole("textbox").getAttribute("aria-invalid")).toBe(
				"true",
			),
		);
		expect(closeMock).toHaveBeenCalledTimes(1);

		const detail = { ID: 7, status: "closed" } as never;
		closeMock.mockResolvedValueOnce({
			success: true,
			code: 200,
			message: "",
			data: detail,
		});
		expect(counted().value).toBe("48.5");
		fireEvent.change(screen.getByRole("textbox"), {
			target: { value: "Faltan 1.50 de cambio" },
		});
		submit();
		await waitFor(() =>
			expect(closeMock).toHaveBeenLastCalledWith(7, {
				counted_cash: 48.5,
				note: "Faltan 1.50 de cambio",
			}),
		);
		await waitFor(() => expect(onClosed).toHaveBeenCalledWith(detail));
	});
});
