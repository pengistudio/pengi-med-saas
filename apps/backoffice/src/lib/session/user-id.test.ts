import { describe, expect, it } from "vitest";
import { userIdOf } from "./user-id";

const jwt = (payload: object) => `h.${btoa(JSON.stringify(payload))}.s`;

describe("userIdOf", () => {
	it("reads the userId claim", () => {
		expect(userIdOf(jwt({ userId: 7 }))).toBe(7);
	});
	it("is undefined without a token or with an unreadable one", () => {
		expect(userIdOf(undefined)).toBeUndefined();
		expect(userIdOf("garbage")).toBeUndefined();
		expect(userIdOf(jwt({ userId: "7" }))).toBeUndefined();
	});
});
