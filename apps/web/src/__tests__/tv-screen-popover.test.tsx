import { act, fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@pengi/shared", async (importOriginal) => ({
	...(await importOriginal<typeof import("@pengi/shared")>()),
	useText: () => ({ textGet: (key: string) => key }),
}));

const mocks = vi.hoisted(() => ({
	getDisplayToken: vi.fn(),
	generateDisplayToken: vi.fn(),
	getDisplayTokenQr: vi.fn(),
}));
vi.mock("@/api/clinical-service", () => mocks);

import {
	DisplayQr,
	TvScreenPopover,
} from "@/pages/clincal/waiting-room/tv-screen-popover";

const TOKEN = "Ab3_-xYz0123456789abcdefghijKLMN";
const NEW_TOKEN = "Zz9_-xYz0123456789abcdefghijKLMN";
// FRONTEND_URL on the API: deliberately not window.location.origin.
const linkFor = (token: string) =>
	`https://app.example.com/display/waiting-room?token=${token}`;
const png = () => new Blob(["png"], { type: "image/png" });

describe("TvScreenPopover", () => {
	const create = vi.fn();
	const revoke = vi.fn();

	beforeEach(() => {
		for (const m of Object.values(mocks)) m.mockReset();
		create.mockReset();
		revoke.mockReset();
		let n = 0;
		create.mockImplementation(() => `blob:qr-${++n}`);
		window.URL.createObjectURL = create;
		window.URL.revokeObjectURL = revoke;
		mocks.getDisplayToken.mockResolvedValue({
			success: true,
			data: { token: TOKEN, display_url: linkFor(TOKEN) },
		});
		mocks.getDisplayTokenQr.mockResolvedValue({ success: true, data: png() });
	});

	it("shows the QR of the TV link next to the link when opened", async () => {
		render(<TvScreenPopover />);
		fireEvent.click(
			screen.getByRole("button", { name: /waiting_room.tv.title/ }),
		);

		const img = await screen.findByAltText("waiting_room.tv.qr_alt");
		expect(img).toHaveAttribute("src", "blob:qr-1");
		expect(screen.getByText("waiting_room.tv.qr_hint")).toBeInTheDocument();
		expect(screen.getByText(linkFor(TOKEN))).toBeInTheDocument();
		expect(mocks.getDisplayTokenQr).toHaveBeenCalledTimes(1);
	});

	it("copies the link the API returned, the same one the QR encodes", async () => {
		const writeText = vi.fn().mockResolvedValue(undefined);
		Object.defineProperty(navigator, "clipboard", {
			value: { writeText },
			configurable: true,
		});
		render(<TvScreenPopover />);
		fireEvent.click(
			screen.getByRole("button", { name: /waiting_room.tv.title/ }),
		);
		await screen.findByText(linkFor(TOKEN));

		fireEvent.click(
			screen.getByRole("button", { name: "waiting_room.tv.copy_code" }),
		);
		await vi.waitFor(() =>
			expect(writeText).toHaveBeenCalledWith(linkFor(TOKEN)),
		);
	});

	it("refreshes the QR after regenerating the token and revokes the old one", async () => {
		mocks.generateDisplayToken.mockResolvedValue({
			success: true,
			data: { token: NEW_TOKEN, display_url: linkFor(NEW_TOKEN) },
		});
		render(<TvScreenPopover />);
		fireEvent.click(
			screen.getByRole("button", { name: /waiting_room.tv.title/ }),
		);
		await screen.findByAltText("waiting_room.tv.qr_alt");

		fireEvent.click(screen.getByText("waiting_room.tv.regenerate"));
		fireEvent.click(screen.getByText("waiting_room.tv.regenerate_confirm"));

		await vi.waitFor(() =>
			expect(screen.getByAltText("waiting_room.tv.qr_alt")).toHaveAttribute(
				"src",
				"blob:qr-2",
			),
		);
		expect(mocks.getDisplayTokenQr).toHaveBeenCalledTimes(2);
		expect(revoke).toHaveBeenCalledWith("blob:qr-1");
		expect(screen.getByText(linkFor(NEW_TOKEN))).toBeInTheDocument();
	});
});

describe("DisplayQr", () => {
	const revoke = vi.fn();

	beforeEach(() => {
		mocks.getDisplayTokenQr.mockReset();
		revoke.mockReset();
		window.URL.createObjectURL = vi.fn(() => "blob:qr");
		window.URL.revokeObjectURL = revoke;
	});

	it("revokes the object URL on unmount", async () => {
		mocks.getDisplayTokenQr.mockResolvedValue({ success: true, data: png() });
		const { unmount } = render(<DisplayQr />);
		expect(
			await screen.findByAltText("waiting_room.tv.qr_alt"),
		).toHaveAttribute("src", "blob:qr");
		expect(revoke).not.toHaveBeenCalled();

		unmount();
		expect(revoke).toHaveBeenCalledWith("blob:qr");
	});

	it("renders nothing when the QR cannot be loaded", async () => {
		mocks.getDisplayTokenQr.mockResolvedValue({ success: false });
		const { container } = render(<DisplayQr />);
		await act(async () => {});
		expect(container).toBeEmptyDOMElement();
		expect(window.URL.createObjectURL).not.toHaveBeenCalled();
	});
});
