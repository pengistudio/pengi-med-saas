import { type Doctor, SPECIALTY_OTHER } from "@/api/doctors-service";
import type { DoctorFormValues } from "@/sections/forms/doctors/doctor-form";

type TextGet = (key: string) => string;

/** A doctor's specialty for display: the catalog label, or the free name for "other". */
export function specialtyLabel(
	doctor: Pick<Doctor, "specialty" | "specialty_other">,
	textGet: TextGet,
) {
	if (doctor.specialty === SPECIALTY_OTHER && doctor.specialty_other) {
		return doctor.specialty_other;
	}
	return textGet(`doctors.specialty.${doctor.specialty}`);
}

/** Inline styles that paint an agenda block with a doctor's #RRGGBB color. */
export function doctorColorStyle(color: string) {
	return {
		backgroundColor: `${color}26`, // ~15% alpha, like the palette's bg-…/15
		borderColor: color,
		color,
	};
}

/**
 * The profile fields of a submitted form, as the API expects them. An empty
 * color is left out: on create the backend picks one, on update it keeps it.
 */
export function toProfilePayload(values: DoctorFormValues) {
	return {
		full_name: values.full_name.trim(),
		specialty: values.specialty,
		specialty_other:
			values.specialty === SPECIALTY_OTHER
				? (values.specialty_other ?? "").trim()
				: "",
		id_number: values.id_number ?? "",
		professional_registry: values.professional_registry ?? "",
		phone: values.phone ?? "",
		email: values.email ?? "",
		...(values.color ? { color: values.color } : {}),
	};
}
