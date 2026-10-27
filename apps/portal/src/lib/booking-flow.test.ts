import type { ErrorResponse } from "@pengi/shared";
import { describe, expect, it } from "vitest";
import type { BookingOptions } from "@/api/portal-service";
import {
	bookingErrorKind,
	canBook,
	currentStep,
	initialSelection,
	providersOf,
	selectProvider,
	slotRange,
	stepBack,
	typesFor,
	visibleSteps,
} from "./booking-flow";

const consult = { id: 1, name: "Consulta", duration_minutes: 30 };
const control = { id: 2, name: "Control", duration_minutes: 15 };
const ana = {
	id: 10,
	full_name: "Ana Paz",
	specialty: "pediatrics",
	specialty_other: "",
	type_ids: [1, 2],
};
const luis = {
	id: 11,
	full_name: "Luis Mora",
	specialty: "cardiology",
	specialty_other: "",
	type_ids: [1],
};

const options = (over: Partial<BookingOptions> = {}): BookingOptions => ({
	choose_by: "doctor",
	min_date: "2026-10-27",
	max_date: "2026-12-25",
	doctors: [ana, luis],
	specialties: [
		{ specialty: "pediatrics", specialty_other: "", type_ids: [1, 2] },
		{ specialty: "cardiology", specialty_other: "", type_ids: [1] },
	],
	types: [consult, control],
	...over,
});

const error = (code: number, errorCode = "E-X"): ErrorResponse => ({
	success: false,
	code,
	message: "msg",
	data: { error_code: errorCode, error_message: "msg" },
});

describe("booking flow", () => {
	it("asks for everything when there are several options", () => {
		const o = options();
		const selection = initialSelection(o);
		expect(selection).toEqual({});
		expect(currentStep(selection)).toBe("provider");
		expect(visibleSteps(o, selection)).toEqual([
			"provider",
			"type",
			"when",
			"review",
		]);
	});

	it("skips the provider and type steps when each has one option", () => {
		const o = options({ doctors: [luis] });
		const selection = initialSelection(o);
		expect(selection.provider?.key).toBe("d:11");
		expect(selection.type).toEqual(consult);
		expect(currentStep(selection)).toBe("when");
		expect(visibleSteps(o, selection)).toEqual(["when", "review"]);
		// Going back from the first visible step leaves the flow.
		expect(stepBack(o, selection)).toBeUndefined();
	});

	it("picks the type with the doctor when the doctor offers only one", () => {
		const o = options();
		const luisProvider = providersOf(o)[1];
		const selection = selectProvider(o, luisProvider);
		expect(selection.type).toEqual(consult);
		expect(visibleSteps(o, selection)).toEqual(["provider", "when", "review"]);
		// Back from the times goes to the doctors, not to a skipped type step.
		expect(stepBack(o, selection)).toEqual({});
	});

	it("goes back from the review to the times, keeping the day", () => {
		const o = options();
		const selection = {
			...selectProvider(o, providersOf(o)[0]),
			type: control,
			date: "2026-10-28",
			time: "09:00",
		};
		expect(currentStep(selection)).toBe("review");
		expect(stepBack(o, selection)).toEqual({
			...selection,
			time: undefined,
		});
	});

	it("offers a specialty the types of its doctors", () => {
		const o = options({ choose_by: "specialty" });
		const providers = providersOf(o);
		expect(providers.map((p) => p.key)).toEqual([
			"s:pediatrics|",
			"s:cardiology|",
		]);
		expect(typesFor(o, providers[0])).toEqual([consult, control]);
		expect(typesFor(o, providers[1])).toEqual([consult]);
	});

	it("leaves out doctors without bookable types", () => {
		const o = options({ doctors: [{ ...ana, type_ids: [] }, luis] });
		expect(providersOf(o).map((p) => p.key)).toEqual(["d:11"]);
		expect(canBook(options({ doctors: [] }))).toBe(false);
	});

	it("pages the window 14 days at a time", () => {
		expect(slotRange("2026-10-27", "2026-12-25")).toEqual({
			from: "2026-10-27",
			to: "2026-11-09",
		});
		expect(slotRange("2026-12-20", "2026-12-25")).toEqual({
			from: "2026-12-20",
			to: "2026-12-25",
		});
		expect(slotRange("2026-12-26", "2026-12-25")).toBeUndefined();
	});

	it("tells slot taken and too many pending apart", () => {
		expect(bookingErrorKind(error(409))).toBe("slot_taken");
		expect(bookingErrorKind(error(429))).toBe("too_many_pending");
		expect(bookingErrorKind(error(409, "E-PORTAL-011"))).toBe(
			"too_many_pending",
		);
		expect(bookingErrorKind(error(400))).toBe("other");
	});
});
