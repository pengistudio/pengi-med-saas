import { describe, expect, it } from "vitest";
import type { VitalSigns } from "@/api/clinical-service";
import {
	fromVitalSigns,
	toVitalSignsInput,
} from "@/sections/forms/clinical/vital-signs-schema";

describe("vital signs form values", () => {
	it("keeps only the measurements that were taken", () => {
		const stored = {
			ID: 4,
			medical_record_id: null,
			appointment_id: 9,
			weight: 70,
			height: null,
			blood_pressure: "",
			temperature: 36.5,
		} as VitalSigns;

		expect(fromVitalSigns(stored)).toEqual({ weight: 70, temperature: 36.5 });
	});

	it("sends empty measurements as null and an empty pressure as absent", () => {
		expect(toVitalSignsInput({ weight: 70, blood_pressure: "" })).toEqual({
			weight: 70,
			height: null,
			blood_pressure: undefined,
			temperature: null,
			heart_rate: null,
			o2_saturation: null,
		});
	});
});
