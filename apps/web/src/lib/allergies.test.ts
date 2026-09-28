import { describe, expect, it } from "vitest";
import { parseAllergies } from "./allergies";

describe("parseAllergies", () => {
	it("treats missing, empty and blank values as no allergies", () => {
		expect(parseAllergies(undefined)).toEqual([]);
		expect(parseAllergies(null)).toEqual([]);
		expect(parseAllergies("")).toEqual([]);
		expect(parseAllergies("   ")).toEqual([]);
	});

	it("treats an empty JSON array as no allergies", () => {
		expect(parseAllergies("[]")).toEqual([]);
		expect(parseAllergies(" [ ] ")).toEqual([]);
		expect(parseAllergies('["", "  "]')).toEqual([]);
	});

	it("reads a JSON array of strings", () => {
		expect(parseAllergies('["Penicilina", " Polen "]')).toEqual([
			"Penicilina",
			"Polen",
		]);
	});

	it("reads comma-separated text", () => {
		expect(parseAllergies("Penicilina, Polen,, Mariscos ")).toEqual([
			"Penicilina",
			"Polen",
			"Mariscos",
		]);
	});

	it("falls back to comma-separated text when the brackets are not JSON", () => {
		expect(parseAllergies("[Penicilina, Polen")).toEqual([
			"[Penicilina",
			"Polen",
		]);
	});

	it("drops duplicates so each allergy renders once", () => {
		expect(parseAllergies("Polen, Polen")).toEqual(["Polen"]);
	});
});
