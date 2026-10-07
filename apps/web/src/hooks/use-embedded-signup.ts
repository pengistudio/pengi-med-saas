import { useText } from "@pengi/shared";
import { useToast } from "@pengi/ui";
import React from "react";
import {
	connectWhatsAppEmbedded,
	type WhatsAppAccount,
	type WhatsAppConfig,
} from "@/api/whatsapp-service";
import {
	type EmbeddedSignupResult,
	loadFacebookSdk,
	parseEmbeddedSignupMessage,
} from "@/lib/facebook-sdk";

/**
 * How long to wait for the popup's ids once FB.login returned the code. Past
 * it the flow is given up instead of spinning forever.
 */
export const EMBEDDED_SIGNUP_TIMEOUT_MS = 45_000;

/**
 * Runs Meta's Embedded Signup: FB.login returns the code, the popup posts the
 * WABA and phone number ids; once both are in, the account is connected.
 */
export function useEmbeddedSignup(
	config: WhatsAppConfig | null,
	onConnected: (account: WhatsAppAccount) => void,
) {
	const { textGet } = useText();
	const { errorToast } = useToast();
	const [running, setRunning] = React.useState(false);
	const pending = React.useRef<{
		code?: string;
		ids?: EmbeddedSignupResult;
	} | null>(null);
	const timeout = React.useRef<ReturnType<typeof setTimeout> | null>(null);
	const mounted = React.useRef(false);
	// Latest callback, so a new function from the caller doesn't restart the
	// listener (and drop an in-progress flow).
	const onConnectedRef = React.useRef(onConnected);
	React.useLayoutEffect(() => {
		onConnectedRef.current = onConnected;
	});

	const clearWait = React.useCallback(() => {
		if (timeout.current !== null) {
			clearTimeout(timeout.current);
			timeout.current = null;
		}
	}, []);

	const stop = React.useCallback(() => {
		clearWait();
		pending.current = null;
		if (mounted.current) setRunning(false);
	}, [clearWait]);

	const finish = React.useCallback(async () => {
		const state = pending.current;
		if (!state?.code || !state.ids) return;
		clearWait();
		pending.current = null;
		const res = await connectWhatsAppEmbedded({
			code: state.code,
			...state.ids,
		});
		if (!mounted.current) return;
		setRunning(false);
		if (res.success && res.data) onConnectedRef.current(res.data);
	}, [clearWait]);

	React.useEffect(() => {
		mounted.current = true;
		return () => {
			mounted.current = false;
			clearWait();
			pending.current = null;
		};
	}, [clearWait]);

	React.useEffect(() => {
		function onMessage(event: MessageEvent) {
			const parsed = parseEmbeddedSignupMessage(event.origin, event.data);
			if (!parsed || !pending.current) return;
			if (parsed.type === "cancel") {
				stop();
				return;
			}
			pending.current.ids = parsed.result;
			void finish();
		}
		window.addEventListener("message", onMessage);
		return () => window.removeEventListener("message", onMessage);
	}, [finish, stop]);

	const start = React.useCallback(async () => {
		if (!config?.embedded_signup_available) return;
		clearWait();
		setRunning(true);
		pending.current = {};
		let fb: Awaited<ReturnType<typeof loadFacebookSdk>>;
		try {
			fb = await loadFacebookSdk(config.app_id, config.graph_version);
		} catch {
			stop();
			if (mounted.current)
				errorToast(null, textGet("settings.whatsapp.embedded.sdk_error"));
			return;
		}
		fb.login(
			(response) => {
				const code = response.authResponse?.code;
				if (!code || !pending.current) {
					// Closed before finishing.
					stop();
					return;
				}
				pending.current.code = code;
				if (pending.current.ids) {
					void finish();
					return;
				}
				// The ids should follow right away; don't wait forever.
				timeout.current = setTimeout(() => {
					timeout.current = null;
					if (!pending.current) return;
					stop();
					if (mounted.current)
						errorToast(null, textGet("settings.whatsapp.embedded.timeout"));
				}, EMBEDDED_SIGNUP_TIMEOUT_MS);
			},
			{
				config_id: config.config_id,
				response_type: "code",
				override_default_response_type: true,
				extras: { setup: {}, sessionInfoVersion: "3" },
			},
		);
	}, [clearWait, config, errorToast, finish, stop, textGet]);

	return { start, running };
}
