import { describe, expect, it } from "vitest";
import { extractDisplayToken } from "@/pages/display/pair";

const TOKEN = "Ab3_-xYz0123456789abcdefghijKLMN";

describe("extractDisplayToken", () => {
	it("accepts a bare token", () => {
		expect(extractDisplayToken(`  ${TOKEN} `)).toBe(TOKEN);
	});

	it("takes the token out of the waiting-room link", () => {
		expect(
			extractDisplayToken(
				`https://app.example.com/display/waiting-room?token=${TOKEN}`,
			),
		).toBe(TOKEN);
	});

	it("rejects the old 8-digit codes and anything else", () => {
		expect(extractDisplayToken("12345678")).toBeNull();
		expect(extractDisplayToken(`${TOKEN}!`)).toBeNull();
		expect(
			extractDisplayToken("https://app.example.com/display/waiting-room"),
		).toBeNull();
		expect(extractDisplayToken("")).toBeNull();
	});
});
