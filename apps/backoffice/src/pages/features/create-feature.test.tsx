import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "@/test/test-wrapper";

const mocks = vi.hoisted(() => ({ create: vi.fn() }));

vi.mock("@/api/feature-service", () => ({
	features: { create: mocks.create },
}));
vi.mock("@/components/features/permission-picker", () => ({
	PermissionPicker: () => null,
}));

import CreateFeature from "./create-feature";

describe("CreateFeature", () => {
	beforeEach(() => {
		mocks.create.mockReset();
		mocks.create.mockResolvedValue({ success: true, data: {} });
	});

	it("does not submit an empty form", async () => {
		renderWithProviders(<CreateFeature />);

		await userEvent.click(
			screen.getByRole("button", { name: /backoffice\.features\.create/ }),
		);

		await waitFor(() =>
			expect(document.querySelectorAll('[aria-invalid="true"]').length).toBe(2),
		);
		expect(mocks.create).not.toHaveBeenCalled();
	});

	it("submits the code and name once valid", async () => {
		renderWithProviders(<CreateFeature />);

		await userEvent.type(
			screen.getByLabelText(/backoffice\.features\.col\.code/),
			"CLINICAL",
		);
		await userEvent.type(
			screen.getByLabelText(/backoffice\.features\.col\.name/),
			"Clinica",
		);
		await userEvent.click(
			screen.getByRole("button", { name: /backoffice\.features\.create/ }),
		);

		await waitFor(() => expect(mocks.create).toHaveBeenCalledTimes(1));
		expect(mocks.create.mock.calls[0][0]).toMatchObject({
			code: "CLINICAL",
			name: "Clinica",
		});
	});
});
