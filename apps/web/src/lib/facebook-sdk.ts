// Meta's JavaScript SDK, loaded on demand for WhatsApp Embedded Signup.

export interface FacebookLoginResponse {
	status?: string;
	authResponse?: { code?: string } | null;
}

export interface FacebookSdk {
	init: (options: {
		appId: string;
		version: string;
		xfbml?: boolean;
		autoLogAppEvents?: boolean;
	}) => void;
	login: (
		callback: (response: FacebookLoginResponse) => void,
		options: Record<string, unknown>,
	) => void;
}

declare global {
	interface Window {
		FB?: FacebookSdk;
		fbAsyncInit?: () => void;
	}
}

const SDK_URL = "https://connect.facebook.net/en_US/sdk.js";
const SCRIPT_ID = "facebook-jssdk";

let loading: Promise<FacebookSdk> | null = null;
let initializedFor = "";

/** Loads the SDK once and (re)initializes it for `appId`. */
export function loadFacebookSdk(
	appId: string,
	version: string,
): Promise<FacebookSdk> {
	const init = (fb: FacebookSdk) => {
		const key = `${appId}:${version}`;
		if (initializedFor !== key) {
			fb.init({ appId, version, xfbml: false, autoLogAppEvents: true });
			initializedFor = key;
		}
		return fb;
	};

	if (window.FB) return Promise.resolve(init(window.FB));
	if (!loading) {
		loading = new Promise<FacebookSdk>((resolve, reject) => {
			window.fbAsyncInit = () => {
				if (window.FB) resolve(window.FB);
				else reject(new Error("facebook sdk unavailable"));
			};
			if (document.getElementById(SCRIPT_ID)) return;
			const script = document.createElement("script");
			script.id = SCRIPT_ID;
			script.src = SDK_URL;
			script.async = true;
			script.defer = true;
			script.crossOrigin = "anonymous";
			script.onerror = () => {
				loading = null;
				script.remove();
				reject(new Error("facebook sdk failed to load"));
			};
			document.body.appendChild(script);
		});
	}
	return loading.then(init);
}

/** The ids Embedded Signup posts back when the user finishes the flow. */
export interface EmbeddedSignupResult {
	waba_id: string;
	phone_number_id: string;
}

export type EmbeddedSignupEvent =
	| { type: "finish"; result: EmbeddedSignupResult }
	| { type: "cancel" };

function isFacebookOrigin(origin: string) {
	try {
		const host = new URL(origin).hostname;
		return host === "facebook.com" || host.endsWith(".facebook.com");
	} catch {
		return false;
	}
}

/**
 * Reads a window `message` event from the Embedded Signup popup. Returns null
 * for anything that isn't one (other origins, other message types).
 */
export function parseEmbeddedSignupMessage(
	origin: string,
	data: unknown,
): EmbeddedSignupEvent | null {
	if (!isFacebookOrigin(origin)) return null;
	let payload: unknown = data;
	if (typeof data === "string") {
		try {
			payload = JSON.parse(data);
		} catch {
			return null;
		}
	}
	if (!payload || typeof payload !== "object") return null;
	const msg = payload as {
		type?: string;
		event?: string;
		data?: { waba_id?: string; phone_number_id?: string };
	};
	if (msg.type !== "WA_EMBEDDED_SIGNUP") return null;
	if (msg.event === "FINISH") {
		const waba = msg.data?.waba_id;
		const phone = msg.data?.phone_number_id;
		if (!waba || !phone) return null;
		return {
			type: "finish",
			result: { waba_id: String(waba), phone_number_id: String(phone) },
		};
	}
	if (msg.event === "CANCEL") return { type: "cancel" };
	return null;
}
