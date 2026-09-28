import { type AxiosInstance, isAxiosError } from "axios";

export type MessageMap = Record<string, string>;

/**
 * Result of loading the messages of one language:
 * - `fresh`: the server sent them (with the hash of their content, if it
 *   could be read).
 * - `not-modified`: the cached messages sent as `If-None-Match` are current.
 */
export type MessagesResult =
	| { status: "fresh"; messages: MessageMap; etag: string | undefined }
	| { status: "not-modified" };

let messagesClient: AxiosInstance | undefined;

/** Set by initShared: the (unauthenticated) client the messages come from. */
export function setMessagesClient(client: AxiosInstance) {
	messagesClient = client;
}

/**
 * GET /i18n/messages. The client is used directly instead of through
 * HttpService because the response headers (ETag) and a bare 304 matter here.
 * Throws when the request fails.
 *
 * @param etag hash of the cached messages of `lang`; the server answers 304
 *   (empty body) when it still matches.
 */
export async function getMessages(
	lang: string,
	etag?: string,
): Promise<MessagesResult> {
	if (!messagesClient) {
		throw new Error(
			"@pengi/shared: call initShared({ client }) before loading messages",
		);
	}
	const client = messagesClient;
	const request = (ifNoneMatch?: string) =>
		client.get<{ data?: unknown }>("/i18n/messages", {
			params: { lang },
			headers: ifNoneMatch ? { "If-None-Match": ifNoneMatch } : undefined,
			// Because If-None-Match is set by hand, the browser hands the 304
			// through instead of answering from its HTTP cache: accept it.
			validateStatus: (status) =>
				(status >= 200 && status < 300) || status === 304,
		});

	let response: Awaited<ReturnType<typeof request>>;
	try {
		response = await request(etag);
	} catch (error) {
		// No response at all with If-None-Match set: likely a CORS preflight
		// rejecting the header. Retry plainly rather than keep a stale cache.
		if (!etag || !isAxiosError(error) || error.response) throw error;
		response = await request();
	}
	if (response.status === 304) return { status: "not-modified" };

	const header = response.headers.etag;
	return {
		status: "fresh",
		messages: toMessageMap(response.data?.data),
		etag: typeof header === "string" && header ? header : undefined,
	};
}

/** `data` is `{key: value}`; the older `[{key, value}]` shape is accepted too. */
function toMessageMap(data: unknown): MessageMap {
	if (Array.isArray(data)) {
		return Object.fromEntries(
			data.map((m: { key: string; value: string }) => [m.key, m.value]),
		);
	}
	if (data && typeof data === "object") return data as MessageMap;
	throw new Error("@pengi/shared: unexpected /i18n/messages response");
}
