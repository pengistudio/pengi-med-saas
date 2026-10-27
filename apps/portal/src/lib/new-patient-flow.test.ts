import type { ErrorResponse } from "@pengi/shared";
import { describe, expect, it } from "vitest";
import {
	birthDateBounds,
	contactChannels,
	contactErrorKind,
	initialPatientValues,
	isValidBirthDate,
	isValidCedula,
	isValidContact,
	isValidDocument,
	newPatientErrorKind,
	newPatientSchema,
	newPatientSteps,
	normalizeDocument,
	toPatientData,
} from "./new-patient-flow";

const fail = (code: number, errorCode = "E-X"): ErrorResponse => ({
	success: false,
	code,
	message: "msg",
	data: { error_code: errorCode, error_message: "msg" },
});

describe("isValidCedula", () => {
	it.each(["1710034065", "0102030400", "3000000012", "0912345675"])(
		"accepts %s",
		(id) => expect(isValidCedula(id)).toBe(true),
	);

	it.each([
		["wrong check digit", "1710034066"],
		["9 digits", "171003406"],
		["letters", "17100340a5"],
		["province 00", "0012345678"],
		["province 25", "2512345678"],
		["third digit 6", "1760034065"],
	])("rejects %s", (_, id) => expect(isValidCedula(id)).toBe(false));
});

describe("documents", () => {
	it("uppercases passports and strips spaces", () => {
		expect(normalizeDocument("passport", " ab 12-345 ")).toBe("AB12345");
		expect(isValidDocument("passport", "ab123")).toBe(true);
		expect(isValidDocument("passport", "ab12")).toBe(false);
		expect(isValidDocument("passport", "AB-12.34")).toBe(false);
		expect(isValidDocument("cedula", "171003406 5")).toBe(true);
	});
});

describe("isValidBirthDate", () => {
	const today = new Date(2026, 9, 26);
	it("takes past days within 120 years", () => {
		expect(isValidBirthDate("1990-05-01", today)).toBe(true);
		expect(isValidBirthDate("2026-10-25", today)).toBe(true);
		expect(isValidBirthDate(birthDateBounds(today).min, today)).toBe(true);
	});
	it("rejects today, the future, too old and impossible days", () => {
		expect(isValidBirthDate("2026-10-26", today)).toBe(false);
		expect(isValidBirthDate("2030-01-01", today)).toBe(false);
		expect(isValidBirthDate("1900-01-01", today)).toBe(false);
		expect(isValidBirthDate("1990-02-31", today)).toBe(false);
		expect(isValidBirthDate("", today)).toBe(false);
	});
});

describe("contacts", () => {
	it("offers WhatsApp only when the clinic has it", () => {
		expect(contactChannels({ whatsapp: true, email: true })).toEqual([
			"whatsapp",
			"email",
		]);
		expect(contactChannels({ whatsapp: false, email: true })).toEqual([
			"email",
		]);
	});
	it("checks the contact by channel", () => {
		expect(isValidContact("email", "ana@mail.com")).toBe(true);
		expect(isValidContact("email", "ana@mail")).toBe(false);
		expect(isValidContact("whatsapp", "099 123 4567")).toBe(true);
		expect(isValidContact("whatsapp", "+593991234567")).toBe(true);
		expect(isValidContact("whatsapp", "12345")).toBe(false);
	});
});

describe("personal data", () => {
	const verified = { channel: "whatsapp" as const, contact: "0991234567" };
	const values = {
		...initialPatientValues(verified),
		first_name: " Ana María ",
		last_name: "Paz",
		document: "1710034065",
		birth_date: "1990-05-01",
	};

	it("prefills the verified contact", () => {
		expect(initialPatientValues(verified).phone).toBe("0991234567");
		expect(
			initialPatientValues({ channel: "email", contact: "a@b.co" }).email,
		).toBe("a@b.co");
	});

	it("validates with message keys", () => {
		expect(newPatientSchema.safeParse(values).success).toBe(true);
		const bad = newPatientSchema.safeParse({
			...values,
			first_name: " ",
			document: "1710034066",
			email: "nope",
		});
		expect(bad.success).toBe(false);
		const messages = bad.error?.issues.map((i) => i.message);
		expect(messages).toEqual(
			expect.arrayContaining([
				"portal.new_patient.error.first_name",
				"portal.new_patient.error.cedula",
				"portal.new_patient.error.email",
			]),
		);
	});

	it("builds the payload: trimmed, verified contact kept, consent only with a phone", () => {
		expect(
			toPatientData(
				{ ...values, phone: "edited", whatsapp_opt_in: true },
				verified,
			),
		).toEqual({
			first_name: "Ana María",
			last_name: "Paz",
			document_type: "cedula",
			document: "1710034065",
			birth_date: "1990-05-01",
			phone: "0991234567",
			whatsapp_opt_in: true,
		});
		expect(
			toPatientData(
				{ ...values, phone: "", email: "", whatsapp_opt_in: true },
				{ channel: "email", contact: "a@b.co" },
			),
		).toEqual({
			first_name: "Ana María",
			last_name: "Paz",
			document_type: "cedula",
			document: "1710034065",
			birth_date: "1990-05-01",
			email: "a@b.co",
		});
	});
});

describe("steps", () => {
	it("puts the data before the review", () => {
		expect(newPatientSteps(["provider", "type", "when", "review"])).toEqual([
			"provider",
			"type",
			"when",
			"details",
			"review",
		]);
		expect(newPatientSteps(["when", "review"])).toEqual([
			"when",
			"details",
			"review",
		]);
	});
});

describe("errors", () => {
	it("maps the code request errors", () => {
		expect(contactErrorKind(fail(400, "E-PORTAL-005"))).toBe("invalid_contact");
		expect(contactErrorKind(fail(400, "E-PORTAL-008"))).toBe("captcha");
		expect(contactErrorKind(fail(404, "E-PORTAL-015"))).toBe("not_accepted");
		expect(contactErrorKind(fail(429))).toBe("other");
	});
	it("maps the request errors", () => {
		expect(newPatientErrorKind(fail(409, "E-PORTAL-010"))).toBe("slot_taken");
		expect(newPatientErrorKind(fail(409, "E-PORTAL-017"))).toBe(
			"already_pending",
		);
		expect(newPatientErrorKind(fail(429, "E-PORTAL-018"))).toBe("clinic_full");
		expect(newPatientErrorKind(fail(400, "E-PORTAL-016"))).toBe("invalid_data");
		expect(newPatientErrorKind(fail(401, "E-PORTAL-002"))).toBe("expired");
		expect(newPatientErrorKind(fail(500))).toBe("other");
	});
});
