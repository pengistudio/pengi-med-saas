import { describe, expect, it } from "vitest";
import {
	conversationTitle,
	displayPhone,
	mergeMessages,
} from "@/lib/whatsapp-thread";

describe("mergeMessages", () => {
	it("keeps the thread oldest first without duplicates", () => {
		const loaded = [{ id: 3 }, { id: 4 }];
		expect(mergeMessages(loaded, [{ id: 4 }, { id: 5 }])).toEqual([
			{ id: 3 },
			{ id: 4 },
			{ id: 5 },
		]);
		expect(mergeMessages(loaded, [{ id: 1 }, { id: 2 }])).toEqual([
			{ id: 1 },
			{ id: 2 },
			{ id: 3 },
			{ id: 4 },
		]);
	});

	it("replaces stale copies with the fetched ones", () => {
		const merged = mergeMessages(
			[
				{ id: 1, status: "sent" },
				{ id: 2, status: "sent" },
			],
			[{ id: 2, status: "read" }],
		);
		expect(merged).toEqual([
			{ id: 1, status: "sent" },
			{ id: 2, status: "read" },
		]);
	});
});

describe("conversationTitle", () => {
	it("prefers the patient's name, else the number with its +", () => {
		expect(
			conversationTitle({ patient_name: "Ana Pérez", phone: "593991234567" }),
		).toBe("Ana Pérez");
		expect(conversationTitle({ patient_name: "", phone: "593991234567" })).toBe(
			"+593991234567",
		);
		expect(displayPhone("")).toBe("");
	});
});
