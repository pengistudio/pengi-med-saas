import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => {
	const errorToast = vi.fn();
	const textGet = (key: string) => key;
	return {
		errorToast,
		toast: { errorToast },
		text: { textGet },
		connect: vi.fn(),
		login: vi.fn(),
	};
});
vi.mock("@pengi/shared", () => ({ useText: () => mocks.text }));
vi.mock("@pengi/ui", () => ({ useToast: () => mocks.toast }));
vi.mock("@/api/whatsapp-service", () => ({
	connectWhatsAppEmbedded: mocks.connect,
}));
vi.mock("@/lib/facebook-sdk", async (orig) => ({
	...(await orig<typeof import("@/lib/facebook-sdk")>()),
	loadFacebookSdk: () => Promise.resolve({ init: vi.fn(), login: mocks.login }),
}));

import {
	EMBEDDED_SIGNUP_TIMEOUT_MS,
	useEmbeddedSignup,
} from "@/hooks/use-embedded-signup";

const config = {
	app_id: "app",
	config_id: "cfg",
	graph_version: "v23.0",
	embedded_signup_available: true,
};

function postFinish() {
	window.dispatchEvent(
		new MessageEvent("message", {
			origin: "https://www.facebook.com",
			data: {
				type: "WA_EMBEDDED_SIGNUP",
				event: "FINISH",
				data: { waba_id: "W", phone_number_id: "P" },
			},
		}),
	);
}

/** Makes FB.login answer with `code`. */
function loginReturns(code?: string) {
	mocks.login.mockImplementation((cb: (r: unknown) => void) =>
		cb({ authResponse: code ? { code } : null }),
	);
}

describe("useEmbeddedSignup", () => {
	beforeEach(() => {
		vi.useFakeTimers();
		mocks.errorToast.mockReset();
		mocks.connect.mockReset();
		mocks.login.mockReset();
	});
	afterEach(() => vi.useRealTimers());

	it("connects once both the code and the ids arrive", async () => {
		loginReturns("C");
		const account = { connected: true };
		mocks.connect.mockResolvedValue({ success: true, data: account });
		const onConnected = vi.fn();
		const { result } = renderHook(() => useEmbeddedSignup(config, onConnected));

		await act(() => result.current.start());
		expect(result.current.running).toBe(true);
		await act(async () => postFinish());

		expect(mocks.connect).toHaveBeenCalledWith({
			code: "C",
			waba_id: "W",
			phone_number_id: "P",
		});
		expect(onConnected).toHaveBeenCalledWith(account);
		expect(result.current.running).toBe(false);
		// The wait was cleared: no late timeout toast.
		act(() => vi.advanceTimersByTime(EMBEDDED_SIGNUP_TIMEOUT_MS));
		expect(mocks.errorToast).not.toHaveBeenCalled();
	});

	it("gives up when the ids never arrive", async () => {
		loginReturns("C");
		const { result } = renderHook(() => useEmbeddedSignup(config, vi.fn()));

		await act(() => result.current.start());
		expect(result.current.running).toBe(true);
		act(() => vi.advanceTimersByTime(EMBEDDED_SIGNUP_TIMEOUT_MS));

		expect(result.current.running).toBe(false);
		expect(mocks.errorToast).toHaveBeenCalledWith(
			null,
			"settings.whatsapp.embedded.timeout",
		);
		// A late FINISH is ignored.
		await act(async () => postFinish());
		expect(mocks.connect).not.toHaveBeenCalled();
	});

	it("stops when the popup is closed without a code", async () => {
		loginReturns(undefined);
		const { result } = renderHook(() => useEmbeddedSignup(config, vi.fn()));
		await act(() => result.current.start());
		expect(result.current.running).toBe(false);
	});

	it("clears the wait on unmount", async () => {
		loginReturns("C");
		const { result, unmount } = renderHook(() =>
			useEmbeddedSignup(config, vi.fn()),
		);
		await act(() => result.current.start());
		unmount();
		act(() => vi.advanceTimersByTime(EMBEDDED_SIGNUP_TIMEOUT_MS));
		expect(mocks.errorToast).not.toHaveBeenCalled();
	});
});
