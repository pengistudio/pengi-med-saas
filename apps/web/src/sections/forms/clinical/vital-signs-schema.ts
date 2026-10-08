import { z } from "zod";
import type { VitalSigns, VitalSignsInput } from "@/api/clinical-service";

/** Vital signs as typed in a form; shared by the consultation and triage. */
export const vitalSignsSchema = z.object({
	weight: z.coerce.number().positive().optional().nullable(),
	height: z.coerce.number().positive().optional().nullable(),
	blood_pressure: z.string().optional(),
	temperature: z.coerce.number().positive().optional().nullable(),
	heart_rate: z.coerce.number().int().positive().optional().nullable(),
	o2_saturation: z.coerce.number().int().min(0).max(100).optional().nullable(),
});

export type VitalSignsValues = z.infer<typeof vitalSignsSchema>;

/** The fields, in display order, with their unit and placeholder. */
export const VITAL_SIGN_FIELDS = [
	{ name: "weight", unit: "kg", placeholder: "0.0", step: "0.1" },
	{ name: "height", unit: "cm", placeholder: "0.0", step: "0.1" },
	{ name: "blood_pressure", unit: "mmHg", placeholder: "120/80" },
	{ name: "temperature", unit: "°C", placeholder: "36.5", step: "0.1" },
	{ name: "heart_rate", unit: "bpm", placeholder: "70" },
	{ name: "o2_saturation", unit: "%", placeholder: "98" },
] as const satisfies readonly {
	name: keyof VitalSignsValues;
	unit: string;
	placeholder: string;
	step?: string;
}[];

/** Form values to the payload: empty measurements are sent as null. */
export function toVitalSignsInput(values: VitalSignsValues): VitalSignsInput {
	return {
		weight: values.weight ?? null,
		height: values.height ?? null,
		blood_pressure: values.blood_pressure || undefined,
		temperature: values.temperature ?? null,
		heart_rate: values.heart_rate ?? null,
		o2_saturation: values.o2_saturation ?? null,
	};
}

/** Stored vital signs to form values (only the measurements that were taken). */
export function fromVitalSigns(vs: VitalSigns): VitalSignsValues {
	const values: VitalSignsValues = {};
	for (const { name } of VITAL_SIGN_FIELDS) {
		const value = vs[name];
		if (value !== null && value !== undefined && value !== "") {
			(values as Record<string, unknown>)[name] = value;
		}
	}
	return values;
}
