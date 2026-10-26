import { beforeEach, describe, expect, it } from "vitest";
import * as z from "zod";
import { useMessageStore } from "./message-store";
import { updateZodLocale } from "./zod-i18n";

const messageOf = (schema: z.ZodType, input: unknown) =>
	schema.safeParse(input).error?.issues[0]?.message;

describe("zod messages", () => {
	beforeEach(() => {
		useMessageStore.setState({
			messages: { "form.validation.required": "Campo requerido" },
		});
		updateZodLocale("es");
	});

	it("says a missing or null field is required, whatever its type", () => {
		expect(messageOf(z.string(), undefined)).toBe("Campo requerido");
		expect(messageOf(z.number(), undefined)).toBe("Campo requerido");
		expect(messageOf(z.string(), null)).toBe("Campo requerido");
	});

	it("says an empty string is required when it needs one character", () => {
		expect(messageOf(z.string().min(1), "")).toBe("Campo requerido");
	});

	it("keeps zod's message for other issues", () => {
		expect(messageOf(z.string(), 3)).not.toBe("Campo requerido");
		expect(messageOf(z.string().min(3), "")).not.toBe("Campo requerido");
	});

	it("keeps a schema's own message", () => {
		expect(messageOf(z.string({ error: "Propio" }), undefined)).toBe("Propio");
	});

	it("falls back to zod's message before the catalog loads", () => {
		useMessageStore.setState({ messages: {} });
		expect(messageOf(z.string(), undefined)).toMatch(/.+/);
		expect(messageOf(z.string(), undefined)).not.toBe("Campo requerido");
	});
});
