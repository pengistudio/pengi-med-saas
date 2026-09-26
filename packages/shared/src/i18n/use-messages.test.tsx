import { renderHook, waitFor } from "@testing-library/react";
import axios, { type InternalAxiosRequestConfig } from "axios";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it } from "vitest";
import { initShared } from "..";
import { LanguageProvider } from "./language-context";
import { useMessageStore } from "./message-store";
import { useMessages } from "./use-messages";
import { useText } from "./use-text";

/** A fake API serving `messages`; counts how often they are fetched. */
function serve(messages: Record<string, string>) {
	const calls = { count: 0 };
	initShared({
		client: axios.create({
			adapter: async (config: InternalAxiosRequestConfig) => {
				calls.count++;
				return {
					data: {
						code: 200,
						message: "",
						data: Object.entries(messages).map(([key, value]) => ({
							key,
							value,
						})),
					},
					status: 200,
					statusText: "",
					headers: {},
					config,
				};
			},
		}),
	});
	return calls;
}

const wrapper = ({ children }: { children: ReactNode }) => (
	<LanguageProvider>{children}</LanguageProvider>
);

function useApp(version: string) {
	useMessages(version);
	return useText().textGet;
}

describe("useMessages", () => {
	beforeEach(() => {
		useMessageStore.setState({ messages: {}, version: "", lang: "es" });
	});

	it("loads the messages when there are none", async () => {
		serve({ "plans.title": "Planes" });
		const { result } = renderHook(() => useApp("build-1"), { wrapper });

		await waitFor(() => expect(result.current("plans.title")).toBe("Planes"));
	});

	it("reloads cached messages when the app was rebuilt", async () => {
		useMessageStore.setState({
			messages: { "plans.title": "Planes" },
			version: "build-1",
		});
		const calls = serve({ "plans.title": "Planes", "plans.new": "Nuevo" });

		const { result, rerender } = renderHook(({ v }) => useApp(v), {
			wrapper,
			initialProps: { v: "build-1" },
		});
		expect(result.current("plans.new")).toBe("*plans.new*");
		expect(calls.count).toBe(0);

		rerender({ v: "build-2" });
		await waitFor(() => expect(result.current("plans.new")).toBe("Nuevo"));
		expect(calls.count).toBe(1);
	});
});
