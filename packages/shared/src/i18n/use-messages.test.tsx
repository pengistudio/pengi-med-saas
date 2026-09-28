import { act, renderHook, waitFor } from "@testing-library/react";
import axios, {
	AxiosError,
	AxiosHeaders,
	type AxiosResponse,
	type InternalAxiosRequestConfig,
} from "axios";
import type { ReactNode } from "react";
import { toast } from "sonner";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { initShared } from "..";
import { LanguageProvider, useLanguage } from "./language-context";
import { useMessageStore } from "./message-store";
import { useMessages } from "./use-messages";
import { useText } from "./use-text";

vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));

type Catalog = Record<
	string,
	{ etag: string; messages: Record<string, string> }
>;

/**
 * A fake GET /i18n/messages honoring If-None-Match like the API: 304 with an
 * empty body when the hash matches. Records every request it gets.
 */
function serve(catalog: Catalog | "down", { blockIfNoneMatch = false } = {}) {
	const requests: { lang: string; ifNoneMatch: string | undefined }[] = [];
	initShared({
		client: axios.create({
			adapter: async (config: InternalAxiosRequestConfig) => {
				const lang = config.params?.lang as string;
				const ifNoneMatch =
					(AxiosHeaders.from(config.headers).get("If-None-Match") as
						| string
						| undefined) ?? undefined;
				requests.push({ lang, ifNoneMatch });
				if (blockIfNoneMatch && ifNoneMatch) {
					// What the browser reports when CORS rejects the header.
					throw new AxiosError("Network Error", "ERR_NETWORK", config);
				}

				let response: AxiosResponse;
				if (catalog === "down") {
					response = reply(config, 500, {}, {});
				} else {
					const entry = catalog[lang];
					response =
						ifNoneMatch === entry.etag
							? reply(config, 304, "", { etag: entry.etag })
							: reply(
									config,
									200,
									{ code: 200, message: "", data: entry.messages },
									{ etag: entry.etag, "cache-control": "no-cache" },
								);
				}
				// Custom adapters skip axios' settle(): apply validateStatus here.
				if (!config.validateStatus?.(response.status)) {
					throw new AxiosError(
						"fail",
						"ERR_BAD_RESPONSE",
						config,
						null,
						response,
					);
				}
				return response;
			},
		}),
	});
	return requests;
}

function reply(
	config: InternalAxiosRequestConfig,
	status: number,
	data: unknown,
	headers: Record<string, string>,
): AxiosResponse {
	return {
		data,
		status,
		statusText: "",
		headers: new AxiosHeaders(headers),
		config,
	};
}

const wrapper = ({ children }: { children: ReactNode }) => (
	<LanguageProvider>{children}</LanguageProvider>
);

function useApp() {
	useMessages();
	const { changeLanguage } = useLanguage();
	return { textGet: useText().textGet, changeLanguage };
}

const ES_V1 = { etag: '"es-1"', messages: { "plans.title": "Planes" } };
const ES_V2 = {
	etag: '"es-2"',
	messages: { "plans.title": "Planes", "plans.new": "Nuevo" },
};
const EN_V1 = { etag: '"en-1"', messages: { "plans.title": "Plans" } };

/** What the store looks like after a previous visit cached ES_V1. */
function cacheEsV1() {
	useMessageStore.setState({
		lang: "es",
		messages: ES_V1.messages,
		etag: ES_V1.etag,
	});
}

describe("useMessages", () => {
	beforeEach(() => {
		vi.mocked(toast.error).mockClear();
		useMessageStore.setState({ messages: {}, etag: undefined, lang: "es" });
	});

	it("fetches the messages when there is no cache", async () => {
		const requests = serve({ es: ES_V1, en: EN_V1 });
		const { result } = renderHook(useApp, { wrapper });

		await waitFor(() =>
			expect(result.current.textGet("plans.title")).toBe("Planes"),
		);
		expect(requests).toEqual([{ lang: "es", ifNoneMatch: undefined }]);
		expect(useMessageStore.getState().etag).toBe(ES_V1.etag);
	});

	it("renders the cache right away and keeps it on 304", async () => {
		cacheEsV1();
		const requests = serve({ es: ES_V1, en: EN_V1 });
		const { result } = renderHook(useApp, { wrapper });

		expect(result.current.textGet("plans.title")).toBe("Planes");
		await waitFor(() => expect(requests).toHaveLength(1));
		expect(requests[0]).toEqual({ lang: "es", ifNoneMatch: ES_V1.etag });

		await act(async () => {});
		expect(useMessageStore.getState()).toMatchObject({
			messages: ES_V1.messages,
			etag: ES_V1.etag,
			lang: "es",
		});
		expect(toast.error).not.toHaveBeenCalled();
	});

	it("replaces the cache and its hash when the messages changed", async () => {
		cacheEsV1();
		const requests = serve({ es: ES_V2, en: EN_V1 });
		const { result } = renderHook(useApp, { wrapper });

		expect(result.current.textGet("plans.new")).toBe("*plans.new*");
		await waitFor(() =>
			expect(result.current.textGet("plans.new")).toBe("Nuevo"),
		);
		expect(requests).toEqual([{ lang: "es", ifNoneMatch: ES_V1.etag }]);
		expect(useMessageStore.getState().etag).toBe(ES_V2.etag);
	});

	it("fetches the new language, without the other language's hash", async () => {
		cacheEsV1();
		const requests = serve({ es: ES_V1, en: EN_V1 });
		const { result } = renderHook(useApp, { wrapper });
		await waitFor(() => expect(requests).toHaveLength(1));

		act(() => result.current.changeLanguage("en"));

		await waitFor(() =>
			expect(result.current.textGet("plans.title")).toBe("Plans"),
		);
		expect(requests[1]).toEqual({ lang: "en", ifNoneMatch: undefined });
		expect(useMessageStore.getState()).toMatchObject({
			lang: "en",
			etag: EN_V1.etag,
		});
	});

	it("keeps the cache quietly when revalidation fails", async () => {
		cacheEsV1();
		const requests = serve("down");
		const { result } = renderHook(useApp, { wrapper });

		await waitFor(() => expect(requests).toHaveLength(1));
		await act(async () => {});
		expect(result.current.textGet("plans.title")).toBe("Planes");
		expect(toast.error).not.toHaveBeenCalled();
	});

	it("retries without If-None-Match when the request can't be sent", async () => {
		cacheEsV1();
		const requests = serve(
			{ es: ES_V2, en: EN_V1 },
			{ blockIfNoneMatch: true },
		);
		const { result } = renderHook(useApp, { wrapper });

		await waitFor(() =>
			expect(result.current.textGet("plans.new")).toBe("Nuevo"),
		);
		expect(requests).toEqual([
			{ lang: "es", ifNoneMatch: ES_V1.etag },
			{ lang: "es", ifNoneMatch: undefined },
		]);
	});

	it("accepts the older array-shaped data", async () => {
		initShared({
			client: axios.create({
				adapter: async (config: InternalAxiosRequestConfig) =>
					reply(
						config,
						200,
						{ code: 200, message: "", data: [{ key: "a.b", value: "AB" }] },
						{},
					),
			}),
		});
		const { result } = renderHook(useApp, { wrapper });

		await waitFor(() => expect(result.current.textGet("a.b")).toBe("AB"));
		expect(useMessageStore.getState().etag).toBeUndefined();
	});

	it("reports when there are no messages to show", async () => {
		serve("down");
		renderHook(useApp, { wrapper });

		await waitFor(() => expect(toast.error).toHaveBeenCalledTimes(1));
	});
});
