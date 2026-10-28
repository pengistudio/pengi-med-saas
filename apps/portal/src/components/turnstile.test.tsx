import { act, render, waitFor } from "@testing-library/react";
import React from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { TurnstileHandle } from "./turnstile";

// Cloudflare's API, faked: render() keeps the options so a test can fire the
// widget's callbacks.
function fakeApi() {
	const api = {
		options: undefined as Record<string, unknown> | undefined,
		render: vi.fn((_el: HTMLElement, options: Record<string, unknown>) => {
			api.options = options;
			return "widget-1";
		}),
		reset: vi.fn(),
		remove: vi.fn(),
		fire(name: string, ...args: unknown[]) {
			const callback = api.options?.[name] as
				| ((...a: unknown[]) => void)
				| undefined;
			if (!callback) throw new Error(`no ${name} registered`);
			act(() => callback(...args));
		},
	};
	return api;
}

// A fresh module per test: the script promise is module state.
async function loadTurnstile() {
	vi.resetModules();
	return (await import("./turnstile")).Turnstile;
}

function scripts() {
	return document.head.querySelectorAll<HTMLScriptElement>(
		'script[src^="https://challenges.cloudflare.com/turnstile/"]',
	);
}

function setup(Turnstile: Awaited<ReturnType<typeof loadTurnstile>>) {
	const onToken = vi.fn();
	const onError = vi.fn();
	const ref = React.createRef<TurnstileHandle>();
	const view = render(
		<Turnstile
			siteKey="site-key"
			language="es"
			onToken={onToken}
			onError={onError}
			ref={ref}
		/>,
	);
	return { onToken, onError, ref, ...view };
}

beforeEach(() => {
	delete window.turnstile;
	for (const s of scripts()) s.remove();
});

afterEach(() => {
	delete window.turnstile;
});

describe("Turnstile", () => {
	it("loads Cloudflare's script once and renders the widget with the site key", async () => {
		const Turnstile = await loadTurnstile();
		const api = fakeApi();
		setup(Turnstile);
		setup(Turnstile);
		expect(scripts()).toHaveLength(1);
		const script = scripts()[0];
		expect(script.src).toContain("render=explicit");

		window.turnstile = api;
		act(() => script.dispatchEvent(new Event("load")));

		await waitFor(() => expect(api.render).toHaveBeenCalledTimes(2));
		expect(api.options).toMatchObject({
			sitekey: "site-key",
			language: "es",
			appearance: "interaction-only",
		});
	});

	it("hands over the token and drops it when it expires", async () => {
		const Turnstile = await loadTurnstile();
		const api = fakeApi();
		window.turnstile = api;
		const { onToken } = setup(Turnstile);
		await waitFor(() => expect(api.render).toHaveBeenCalled());

		api.fire("callback", "token-1");
		expect(onToken).toHaveBeenLastCalledWith("token-1");

		api.fire("expired-callback");
		expect(onToken).toHaveBeenLastCalledWith(undefined);
		expect(api.reset).toHaveBeenCalledWith("widget-1");

		api.fire("timeout-callback");
		expect(api.reset).toHaveBeenCalledTimes(2);
	});

	it("reports a widget error", async () => {
		const Turnstile = await loadTurnstile();
		const api = fakeApi();
		window.turnstile = api;
		const { onToken, onError } = setup(Turnstile);
		await waitFor(() => expect(api.render).toHaveBeenCalled());

		api.fire("error-callback");
		expect(onToken).toHaveBeenLastCalledWith(undefined);
		expect(onError).toHaveBeenCalledOnce();
	});

	it("reset() drops the token and asks the widget for a new one", async () => {
		const Turnstile = await loadTurnstile();
		const api = fakeApi();
		window.turnstile = api;
		const { onToken, ref } = setup(Turnstile);
		await waitFor(() => expect(api.render).toHaveBeenCalled());

		api.fire("callback", "token-1");
		act(() => ref.current?.reset());
		expect(onToken).toHaveBeenLastCalledWith(undefined);
		expect(api.reset).toHaveBeenCalledWith("widget-1");
	});

	it("removes the widget on unmount", async () => {
		const Turnstile = await loadTurnstile();
		const api = fakeApi();
		window.turnstile = api;
		const { unmount } = setup(Turnstile);
		await waitFor(() => expect(api.render).toHaveBeenCalled());

		unmount();
		expect(api.remove).toHaveBeenCalledWith("widget-1");
	});

	it("reports a script that fails to load and lets the next mount retry", async () => {
		const Turnstile = await loadTurnstile();
		const first = setup(Turnstile);
		act(() => scripts()[0].dispatchEvent(new Event("error")));
		await waitFor(() => expect(first.onError).toHaveBeenCalledOnce());
		expect(scripts()).toHaveLength(0);

		first.unmount();
		setup(Turnstile);
		expect(scripts()).toHaveLength(1);
	});

	it("reports a script that loads without the API", async () => {
		const Turnstile = await loadTurnstile();
		const { onError } = setup(Turnstile);
		act(() => scripts()[0].dispatchEvent(new Event("load")));
		await waitFor(() => expect(onError).toHaveBeenCalledOnce());
	});
});
