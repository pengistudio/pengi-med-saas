import React from "react";

/**
 * Cloudflare Turnstile widget, explicitly rendered. It hands its token to
 * `onToken` (undefined when the token expires or the widget fails) and
 * `reset()` asks for a new one: tokens work once, so the caller resets after
 * each submit. No dependency: the script is loaded once from Cloudflare.
 */

const SCRIPT_URL =
	"https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit";

interface RenderOptions {
	sitekey: string;
	language?: string;
	theme?: "light" | "dark" | "auto";
	appearance?: "always" | "execute" | "interaction-only";
	callback?: (token: string) => void;
	"expired-callback"?: () => void;
	"error-callback"?: () => void;
	"timeout-callback"?: () => void;
}

interface TurnstileApi {
	render: (container: HTMLElement, options: RenderOptions) => string;
	reset: (widgetId: string) => void;
	remove: (widgetId: string) => void;
}

declare global {
	interface Window {
		turnstile?: TurnstileApi;
	}
}

let scriptPromise: Promise<TurnstileApi> | undefined;

function loadTurnstile(): Promise<TurnstileApi> {
	if (window.turnstile) return Promise.resolve(window.turnstile);
	if (!scriptPromise) {
		scriptPromise = new Promise((resolve, reject) => {
			const script = document.createElement("script");
			script.src = SCRIPT_URL;
			script.async = true;
			script.onload = () =>
				window.turnstile
					? resolve(window.turnstile)
					: reject(new Error("turnstile not loaded"));
			script.onerror = () => {
				scriptPromise = undefined; // let a later mount try again
				script.remove();
				reject(new Error("turnstile not loaded"));
			};
			document.head.appendChild(script);
		});
	}
	return scriptPromise;
}

export interface TurnstileHandle {
	reset: () => void;
}

export function Turnstile({
	siteKey,
	language,
	onToken,
	onError,
	ref,
}: {
	siteKey: string;
	language: string;
	onToken: (token: string | undefined) => void;
	onError: () => void;
	ref?: React.Ref<TurnstileHandle>;
}) {
	const container = React.useRef<HTMLDivElement>(null);
	const widget = React.useRef<{ api: TurnstileApi; id: string }>(undefined);
	// Latest callbacks, so the widget is not re-rendered when they change.
	const callbacks = React.useRef({ onToken, onError });
	callbacks.current = { onToken, onError };

	React.useImperativeHandle(ref, () => ({
		reset: () => {
			callbacks.current.onToken(undefined);
			if (widget.current) widget.current.api.reset(widget.current.id);
		},
	}));

	React.useEffect(() => {
		let cancelled = false;
		loadTurnstile()
			.then((api) => {
				if (cancelled || !container.current) return;
				const id = api.render(container.current, {
					sitekey: siteKey,
					language,
					theme: "light",
					appearance: "interaction-only",
					callback: (token) => callbacks.current.onToken(token),
					"expired-callback": () => {
						callbacks.current.onToken(undefined);
						api.reset(id);
					},
					"timeout-callback": () => {
						callbacks.current.onToken(undefined);
						api.reset(id);
					},
					"error-callback": () => {
						callbacks.current.onToken(undefined);
						callbacks.current.onError();
					},
				});
				widget.current = { api, id };
			})
			.catch(() => {
				if (!cancelled) callbacks.current.onError();
			});
		return () => {
			cancelled = true;
			if (widget.current) {
				widget.current.api.remove(widget.current.id);
				widget.current = undefined;
			}
		};
	}, [siteKey, language]);

	return <div ref={container} />;
}
